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
   (doctor)      (Host, ...)                          (plan.Plan)        (pendiente)
```

## 2. Mapa del repositorio

```
cmd/caratcore/        el programa (la CLI). Solo lee argumentos y llama al núcleo
  main.go             elige el comando: doctor, install, help
  install.go          `install docker --dry-run [--json]`
internal/             el núcleo. "internal" = otros proyectos no pueden importarlo
  doctor/             diagnóstico de solo lectura
  osrelease/          lee /etc/os-release (qué distribución y versión es)
  plan/               el plan de instalación como dato
  docker/             detecta Docker y genera el plan para instalarlo
docs/                 decisiones y esta guía
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
2. instalar `ca-certificates` y `curl`
3. crear `/etc/apt/keyrings`
4. descargar la clave GPG de Docker
5. hacerla legible
6. escribir `/etc/apt/sources.list.d/docker.sources` (el repositorio)
7. `apt-get update` otra vez
8. instalar `docker-ce`, `docker-ce-cli`, `containerd.io`,
   `docker-buildx-plugin` y `docker-compose-plugin`

Y se niega (con un mensaje que explica qué hacer) cuando:
- no es Debian 13 amd64;
- hay paquetes en conflicto (`docker.io`, `containerd`, `runc`...): la guía
  oficial los borra, CaratCore **nunca borra nada por su cuenta**;
- Docker está instalado pero Compose no: no tocamos instalaciones existentes.

Si Docker y Compose ya están, el plan sale vacío ("Nothing to do").

Por qué no `curl get.docker.com | sh`: ejecuta un script remoto sin que veas
cada paso. Es justo lo contrario de lo que quiere CaratCore.

## 5. Cómo se prueba

```
gofmt -l .      # archivos mal formateados (debe salir vacío)
go vet ./...    # errores típicos que compilan pero están mal
go test ./...   # todos los tests
```

Los tests nunca tocan el sistema: usan `Env` falsos (doctor) o `Host`
inventados (docker). La prueba real del instalador se hará solo en una VM
Debian limpia, nunca en Franc.

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

## 7. Qué falta y hacia dónde va

- **Ejecutar el plan**: un paquete ejecutor que pida confirmación, use `sudo`
  si hace falta y pare si un paso falla.
- **Servicios**: AdGuard Home y Uptime Kuma con Docker Compose, con su
  catálogo en JSON incrustado (`go:embed`).
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
