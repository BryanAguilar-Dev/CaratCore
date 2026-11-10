# Guía de estudio de CaratCore

Este documento explica cómo está hecho CaratCore y por qué, para aprender Go
con el propio proyecto. Las decisiones formales y su fecha están en
[decisions.md](decisions.md); aquí está la explicación. Se actualiza al cerrar
cada bloque de trabajo.

## 1. La idea en una frase

CaratCore mira tu máquina, decide qué hay que hacer, te lo enseña paso a paso
con el motivo de cada paso y, solo si lo apruebas, lo ejecuta.

```
system check -> system model -> recommendation -> installation plan -> deployment
   (doctor)      (Host, ...)                          (plan.Plan)         (runner)
```

## 2. Mapa del repositorio

```
cmd/caratcore/        el programa (la CLI). Solo lee argumentos y llama al núcleo
  main.go             elige el comando: doctor, install, uninstall, help
  install.go          `install docker [--dry-run [--json]]`: plan, confirmación, ejecución
  uninstall.go        `uninstall <servicio> [--purge]`: mismo flujo, sin Docker
internal/             el núcleo. "internal" = otros proyectos no pueden importarlo
  doctor/             diagnóstico de solo lectura
  osrelease/          lee /etc/os-release (qué distribución y versión es)
  plan/               el plan de instalación como dato
  docker/             detecta Docker y genera el plan para instalarlo
    docker.asc        clave de firma de Docker, incrustada en el binario
  runner/             ejecuta un plan paso a paso (con sudo si hace falta)
  catalog/            los servicios disponibles (services.json incrustado)
  service/            genera los planes que despliegan y quitan un servicio
  ports/              qué puertos están ocupados (lee /proc/net)
docs/                 decisiones, esta guía y las listas de prueba de cada versión
go.mod                nombre del módulo y versión de Go; sin dependencias externas
.gitattributes        fuerza finales de línea LF (estilo Linux)
```

Regla: **un paquete, una responsabilidad**. La CLI depende del núcleo, nunca
al revés. Así mañana podría haber una interfaz web usando los mismos paquetes.

## 3. Conceptos de Go que aparecen en el proyecto

**Paquetes e importaciones.** Cada carpeta es un paquete. Lo que empieza por
mayúscula (`Render`, `Plan`) es público; en minúscula (`shellQuote`) es privado
del paquete. No hay `public`/`private`: lo decide la primera letra.

**Structs.** Agrupan datos: `type Step struct { ID string; ... }`. No hay
clases; los métodos se cuelgan de un tipo: `func (p Plan) Validate() error`.
`(p Plan)` es el "receptor", como `this` en otros lenguajes.

**Errores como valores.** Go no tiene excepciones. Una función que puede
fallar devuelve `(resultado, error)` y quien llama comprueba
`if err != nil { ... }`. `fmt.Errorf("step %d: %w", i, err)` envuelve un error
añadiendo contexto; `%w` permite luego inspeccionar el error original con
`errors.As` (lo usamos en `detect.go` para leer el código de salida de
`dpkg-query`).

**Slices y maps.** `[]string` es una lista que crece; `map[string]bool` es un
diccionario. `append` añade elementos. `make(map[string]bool, n)` reserva
espacio de antemano.

**Punteros para "opcional".** `File *File` en un paso: si es `nil`, el paso no
escribe archivos. Es la forma sencilla de decir "puede no estar".

**Etiquetas JSON.** `` `json:"id,omitempty"` `` le dice a `encoding/json` cómo
se llama el campo en JSON y que lo omita si está vacío. Así `--json` funciona
sin escribir código de conversión.

**Interfaces pequeñas.** `Render(w io.Writer, ...)` acepta cualquier cosa en la
que se pueda escribir: la terminal (`os.Stdout`), un archivo o un
`bytes.Buffer` en los tests. Por eso los tests pueden comprobar la salida.

**`context.Context`.** Lleva un límite de tiempo (`WithTimeout(..., 20s)`).
Los comandos externos lanzados con `exec.CommandContext` se cancelan si se
pasa del tiempo, así CaratCore nunca se queda colgado.

**Archivos por plataforma.** `disk_linux.go` solo se compila en Linux y
`disk_other.go` en el resto. Go lo decide por el nombre del archivo
(o por una línea `//go:build`). Así el proyecto compila en Windows aunque lea
datos que solo existen en Linux.

**Tests.** Archivos `*_test.go`, funciones `TestXxx(t *testing.T)`. El patrón
más usado aquí es la *tabla de casos*: una lista de entradas y resultados
esperados recorrida con `t.Run`, que da un subtest por caso.

## 4. Los paquetes, uno a uno

### doctor (solo lectura)
Hace comprobaciones (plataforma, CPU, memoria, disco, privilegios, Docker,
puertos, internet, Tailscale) y devuelve un `Report` con un `Check` por cada
una. Cada `Check` tiene un `Status` (OK, INFO, WARN, FAIL); el peor decide el
veredicto y el código de salida.

Truco clave: el struct `Env` contiene *funciones* para leer archivos, ejecutar
comandos o abrir conexiones. El programa real usa las de verdad
(`DefaultEnv`); los tests pasan versiones falsas. Esto se llama *inyección de
dependencias* y permite probar `doctor` en Windows sin un Debian real.

Lee `/proc/meminfo` y `/proc/net/*` en lugar de la salida de `free` o `ss`
porque esa salida cambia con el idioma del sistema y `ss -p` necesita root.

Un puerto ocupado por un servicio que CaratCore ya desplegó (existe su
`compose.yaml`) no es un problema: sale OK. `doctor` no puede comprobar que
el puerto lo ocupe de verdad ese contenedor, porque eso exige `sudo docker` y
`doctor` nunca usa sudo (D24).

Para saber si el usuario podrá usar sudo sin ejecutarlo, `doctor` busca el
GID del grupo `sudo` en `/etc/group` y mira si está entre los grupos del
proceso (`os.Getgroups`). En Debian, pertenecer a ese grupo es lo que da
permiso por defecto. Es una pista, no una garantía (D27).

### osrelease
Una sola función: `Parse` convierte las líneas `CLAVE=valor` de
`/etc/os-release` en un map. Antes vivía dentro de `doctor`; al necesitarla
también `docker`, se movió a su propio paquete para no duplicar código.

### plan (el corazón del proyecto)
Un `Plan` es una lista de `Step`. Cada paso tiene:

| Campo     | Para qué |
|-----------|----------|
| `ID`      | nombre único, para logs y en el futuro para recordar qué se hizo |
| `Title`   | qué hace, en pocas palabras |
| `Reason`  | por qué hace falta. **Obligatorio**: nada se aprueba sin motivo |
| `Command` | programa y argumentos, *o bien*... |
| `File`    | ...un archivo a escribir (ruta, contenido, permisos) |
| `Root`    | si necesita privilegios de administrador |

Ideas importantes:
- **Un plan es solo un dato.** Crearlo o imprimirlo nunca ejecuta nada. Por
  eso `--dry-run` es trivial: imprimir y no ejecutar.
- **`Command` es `[]string`, no un texto.** Se ejecutará sin shell
  (`exec.Command(cmd[0], cmd[1:]...)`). Un argumento con `;` o `$` no puede
  convertirse en otro comando: lo que ves es exactamente lo que se ejecuta.
- **Escribir archivos es un tipo de paso propio**, para no necesitar
  `sh -c "echo ... > archivo"`.
- **El plan no decide si usar `sudo`**: solo marca `Root`. Quien ejecute sabrá
  si ya es root.
- `Validate()` rechaza planes mal formados (sin motivo, sin comando, con ruta
  relativa, IDs repetidos...). Es una red de seguridad contra errores nuestros.
- `ShellJoin` muestra el comando con comillas de shell correctas, para que el
  usuario pueda leerlo o copiarlo.

### docker
Dos mitades separadas a propósito:
- `Detect(ctx)` **lee** la máquina real y rellena un `Host` (distribución,
  versión, arquitectura, si hay Docker y Compose, paquetes en conflicto).
- `InstallPlan(host)` es una **función pura**: con el mismo `Host` siempre
  devuelve el mismo plan, sin tocar nada. Por eso se puede probar con un
  "Debian simulado" en los tests.

`InstallPlan` sigue la guía oficial de Docker para Debian:
1. `apt-get update`
2. instalar `ca-certificates`
3. crear `/etc/apt/keyrings`
4. escribir la clave GPG de Docker (0644, legible para apt)
5. escribir `/etc/apt/sources.list.d/docker.sources` (el repositorio)
6. `apt-get update` otra vez
7. instalar `docker-ce`, `docker-ce-cli`, `containerd.io`,
   `docker-buildx-plugin` y `docker-compose-plugin`

**La clave va dentro del binario** (`docker.asc` con `//go:embed`), en vez de
descargarse con curl como hace la guía. La clave es lo que permite a apt
confiar en los paquetes: si alguien la cambiara al descargarla, apt aceptaría
paquetes falsos. Incrustada, se revisa en el código y `key_test.go` calcula su
huella y la compara con la que publica Docker. La huella OpenPGP v4 es el
SHA-1 de `0x99`, la longitud del paquete de clave pública y su contenido; se
calcula con la librería estándar, sin `gpg` (D26).

Y se niega (con un mensaje que explica qué hacer) cuando:
- no es Debian 13 amd64;
- hay paquetes en conflicto (`docker.io`, `containerd`, `runc`...): la guía
  oficial los borra, CaratCore **nunca borra nada por su cuenta**;
- Docker está instalado pero Compose no: no tocamos instalaciones existentes.

Si Docker y Compose ya están, el plan sale vacío ("Nothing to do").

Por qué no `curl get.docker.com | sh`: ejecuta un script remoto sin que veas
cada paso. Es justo lo contrario de lo que quiere CaratCore.

### runner (ejecutar el plan)
`runner.New().Run(ctx, plan)` ejecuta los pasos en orden. Lo que hace:
- **Valida el plan otra vez** antes de nada. Es barato y evita ejecutar un
  plan roto aunque alguien se olvide de validarlo antes.
- **sudo solo donde hace falta.** Si CaratCore no es root (`os.Geteuid() != 0`),
  los pasos con `Root: true` se ejecutan como `sudo -- comando...`. El `--`
  le dice a sudo "lo que sigue ya no son opciones tuyas".
- **`sudo -v` primero.** Pide la contraseña una vez antes del primer paso. Si
  el usuario no puede usar sudo, se entera sin que se haya tocado nada.
- **Archivos sin shell.** Para un paso `File` escribe el contenido en un
  temporal (`os.CreateTemp`, permisos 0600) y lo copia con
  `install -m 0644 -- temporal destino`. `defer os.Remove(tmp)` borra el
  temporal al salir de la función, pase lo que pase.
- **Para en el primer fallo** y devuelve un `*StepError` con el número del
  paso. La CLI lo reconoce con `errors.As` y explica qué se hizo y qué no.
  No hay rollback: los pasos de Docker se pueden repetir, así que basta con
  volver a lanzar el comando.
- **Ctrl+C con cuidado.** Pulsar Ctrl+C manda SIGINT a todo el grupo de
  procesos de la terminal: a CaratCore y al `apt` que esté corriendo. `apt`
  sabe pararse limpio con SIGINT. CaratCore usa `signal.NotifyContext` para no
  morir: el contexto se cancela y el bucle no empieza el siguiente paso. Por
  eso el comando se lanza con `exec.Command` y **no** con
  `exec.CommandContext`, que lo mataría con SIGKILL (y matar `dpkg` a mitad
  puede romper la base de datos de paquetes).

Cómo se prueba sin ejecutar nada: `Runner` tiene un campo privado
`exec func(args []string) error`. `New()` pone la función real; los tests
ponen una que solo apunta los comandos y puede fingir fallos. Es la misma
idea que `Env` en `doctor`, a escala de una sola función.

Confirmar es cosa de la CLI (`confirm` en `install.go`), no de `runner`: el
núcleo no sabe si hay una persona delante. Recibe `io.Reader`/`io.Writer`, así
que también se podría probar con un `strings.Reader`.

### catalog (qué servicios hay)
`services.json` describe cada servicio: imagen, puertos, carpetas de datos y
qué hacer después de instalar. Va **dentro del binario** gracias a
`//go:embed services.json`: Go copia el archivo en una variable `[]byte` al
compilar, así que el usuario sigue teniendo un solo ejecutable. Añadir un
servicio es editar datos, no código.

Al leerlo, `validate()` rechaza lo peligroso: imágenes sin versión o con
`latest`, IDs con espacios o mayúsculas (se convierten en rutas), textos con
saltos de línea o `#` (romperían el YAML). El test
`TestEmbeddedCatalogIsValid` hace que un `services.json` mal editado no pase
CI.

Por qué JSON y no YAML: Go trae `encoding/json` de serie; para leer YAML
haría falta una librería externa.

### service (desplegar un servicio)
Mismo patrón que `docker`: `Detect` lee la máquina (¿hay Docker y Compose?
¿qué puertos están ocupados? ¿existe ya el `compose.yaml`?) y `DeployPlan`
es una función pura que devuelve el plan.

El `compose.yaml` se genera con `text/template`, la librería de plantillas de
Go: `{{.Service.Image}}` inserta un campo y `{{range .Service.Ports}}` repite
un bloque por cada puerto. `{{-` borra el salto de línea anterior, para que
el resultado quede limpio. Dentro de un `range`, `$` es el dato raíz: así
`{{$.Dir}}` llega a la carpeta del servicio.

La parte más sutil es **repetir un plan**. Si existe un `compose.yaml`:
- idéntico al que CaratCore generaría: es nuestro, se puede repetir (y no se
  comprueban puertos, porque probablemente los ocupa el propio servicio);
- distinto: lo cambió el usuario u otra versión, y CaratCore no lo toca.

Comparar el contenido entero es más sencillo y más seguro que guardar un
"estado" aparte, que todavía no existe.

**Desinstalar** (`remove.go`, D30) reutiliza el mismo `Detect` (con un dato
nuevo, `DirExists`) y otra función pura, `RemovePlan`. Las decisiones están
en qué pasos salen según el estado:
- hay `compose.yaml`: paso `stop` (`docker compose down`), aunque el archivo
  esté editado, porque parar no modifica nada;
- con `--purge` y la carpeta existe: paso `delete-data` (`rm -r --`). El
  `--` impide que una ruta empezando por `-` se lea como opción; la ruta sale
  del catálogo, cuyo `id` ya está validado.
- `--purge` con un `compose.yaml` editado: se niega. Borrar lo que CaratCore
  no escribió sería ir contra D23.

La confirmación de `--purge` pide escribir el id del servicio en vez de
`yes`: así un "yes" por reflejo no basta (`confirm` con `word`, en la CLI).
El aviso de DNS de AdGuard es un dato del catálogo (`remove_warning`), no un
`if id == "adguard-home"` en el código.

### ports
`InUse` lee `/proc/net/{tcp,udp}[6]` y devuelve claves como `"53/udp"`.
Antes vivía dentro de `doctor`; al necesitarlo también `service`, se movió a
su paquete (lo mismo que pasó con `osrelease`).

**Por qué nunca el grupo `docker`.** Quien puede hablar con el demonio de
Docker puede arrancar un contenedor con `/` montado y ser root. Meter al
usuario en el grupo es darle root sin contraseña. CaratCore usa
`sudo docker ...` y `doctor` lo trata como normal (INFO).

**Por qué ufw no protege los servicios.** Linux filtra los paquetes en
cadenas. ufw pone sus reglas en `INPUT`, que filtra lo que va dirigido a la
propia máquina. Docker, al publicar un puerto, reescribe el destino en la
tabla `nat` (PREROUTING) hacia la IP del contenedor, así que el paquete pasa
a la cadena `FORWARD` y nunca llega a `INPUT`. Por eso CaratCore lo avisa en
el README en vez de prometer una protección que no da (D25).

## 5. Cómo se prueba

```
gofmt -l .      # archivos mal formateados (debe salir vacío)
go vet ./...    # errores típicos que compilan pero están mal
go test ./...   # todos los tests
```

Los tests nunca tocan el sistema: usan `Env` falsos (doctor) o `Host`
inventados (docker). La prueba real del instalador se hace en Franc, estando
delante del portátil por si se pierde el acceso remoto (D19).

## 6. Cómo llega CaratCore al usuario

Go compila todo en **un único ejecutable**. Con `CGO_ENABLED=0` ni siquiera
usa librerías de C del sistema: el archivo funciona en cualquier Linux amd64
sin instalar nada. Por eso el usuario no necesita Go.

Compilar para otro sistema ("compilación cruzada") es cambiar dos variables:
`GOOS=linux GOARCH=amd64 go build ...` desde Windows produce un binario Linux.

La automatización vive en `.github/workflows/`:
- `ci.yml`: en cada push comprueba formato, vet y tests en los servidores de
  GitHub. Si algo falla, lo ves en la pestaña *Actions*.
- `release.yml`: al subir una etiqueta (`git tag v0.1.0` y
  `git push origin v0.1.0`) compila el binario, calcula su SHA-256 y crea una
  release en borrador. Revisas y pulsas "Publish".

La versión se mete en el binario al compilar:
`-ldflags "-X main.version=v0.1.0"` sobrescribe la variable `version` de
`main.go`. Por eso en local `caratcore version` dice `dev`.

`SHA256SUMS` permite al usuario comprobar con `sha256sum -c` que el archivo
descargado es exactamente el publicado.

Una release en borrador solo la ve el dueño del repositorio. Eso permite
etiquetar, probar en una VM el binario exacto del borrador y publicar solo si
pasa (`docs/prueba-v0.1.0.md`).

## 7. Qué falta y hacia dónde va

- v0.1.0 publicada (2026-11-10). Repaso de esa sesión, con ejercicios:
  [repaso-2026-11-09.md](repaso-2026-11-09.md). Queda probar los casos sin sudo y los de
  fallo (`docs/prueba-v0.1.0.md`, apartados C, D y G).

- **Probar en Franc los casos de fallo**: conflictos, un fallo a mitad y
  Ctrl+C durante `apt` (la instalación real y los servicios ya funcionan).
- **v0.2 (D29)**: `uninstall` hecho, falta probarlo en la VM
  (`docs/prueba-v0.2.0.md`); después, el modo guiado (D31).
- **Actualizar servicios** (cambiar la versión de la imagen), después de la
  v0.2.
- Pendientes de seguridad y diseño: ver la sección "Pendientes" de
  [decisions.md](decisions.md).

## 8. Preguntas para estudiar

1. ¿Por qué `InstallPlan` no llama a `Detect` por dentro?
2. ¿Qué pasaría si `Command` fuera un `string` y lo ejecutáramos con `sh -c`?
3. ¿Por qué `Render` recibe un `io.Writer` en lugar de escribir en
   `os.Stdout`?
4. ¿Para qué sirve que un plan vacío sea válido?
5. En `detect.go`, ¿por qué se acepta el código de salida 1 de `dpkg-query`
   pero no otros errores?
6. ¿Por qué `plan.go` usa el paquete `path` y no `path/filepath`?
7. ¿Por qué la release se crea como borrador y no publicada directamente?
8. ¿Qué cambia en el binario si se compila con `CGO_ENABLED=1`?
9. ¿Por qué `runner` usa `exec.Command` y no `exec.CommandContext`?
10. ¿Qué pasaría si `confirm` aceptara una respuesta vacía como "sí"?
11. ¿Por qué el temporal de un paso `File` se crea con permisos 0600?
12. ¿Qué problema evitaría fijar la versión de la imagen en vez de usar
    `latest`?
13. Si relanzas `install adguard-home` con AdGuard ya en marcha, ¿por qué no
    falla por "puerto ocupado"?
14. ¿Por qué `pull` y `up` son dos pasos y no uno?
15. ¿En qué caso `doctor` daría OK a un puerto que en realidad ocupa otro
    programa, y por qué se acepta?
16. Si activas ufw con `deny incoming`, ¿por qué el puerto 3000 sigue
    respondiendo desde otro equipo de la red?
17. Si Docker cambiara su clave GPG, ¿qué fallaría en una instalación con
    CaratCore y qué habría que hacer?
18. ¿Por qué `uninstall` sin `--purge` funciona con un `compose.yaml` editado
    y con `--purge` no?
19. Tras `uninstall` sin `--purge`, ¿qué pasa al volver a hacer `install`?
    ¿Se pierde la configuración de AdGuard Home?
20. ¿Por qué `docker image prune` no basta para borrar la imagen de un
    servicio desinstalado?
