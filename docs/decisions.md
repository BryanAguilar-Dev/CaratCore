# Registro de decisiones

Estado: **Decidida** (confirmada por Bryan), **Propuesta** (sin confirmar
explícitamente) o **Pendiente**.

## Decididas
- **D1. Lenguaje: Go.**
- **D2. Núcleo independiente de la interfaz.** La CLI es solo una interfaz.
- **D3. Flujo:** system check -> system model -> recommendation -> installation
  plan -> deployment.
- **D4. `doctor` es de solo lectura.** Nunca instala ni modifica nada.
- **D5. Primera plataforma: Debian 13 amd64.**
- **D6. Licencia: Apache-2.0** (2026-11-09).
- **D7. Repositorio nuevo y limpio** (2026-11-09): el historial anterior se
  descartó; el prototipo anterior no se conserva.
- **D8. `doctor` usa solo la librería estándar.** Lee `/proc` y
  `/etc/os-release` en lugar de `gopsutil` o de la salida de `free`/`df`, que
  depende del idioma del sistema. Menos dependencias y más fácil de entender.
- **D9. Orden de la v1** (2026-11-09): primero el plan de instalación como
  dato, después instalar Docker como pasos de ese plan y por último los
  servicios. Así instalar Docker tiene motivo por paso y `--dry-run` desde el
  principio, en vez de ser un instalador aparte que luego habría que rehacer.
- **D10. Solo Debian en la v1; Windows solo para desarrollar** (2026-11-09).
  Windows exigiría Docker Desktop/WSL2 (otro instalador, otra red, otros
  permisos) y no encaja con un servidor siempre encendido que usa el puerto 53.
  Para no cerrar la puerta: el plan es un dato neutro, cada plataforma tiene su
  generador de pasos, y el código dependiente del sistema va en archivos por
  plataforma (`_linux.go` / `_other.go`). En Windows se compila y se pasan los
  tests (que generan y validan el plan de un Debian simulado); en Windows
  `install --dry-run` responde "plataforma no soportada". La ejecución real,
  solo en Debian.
- **D11. Forma del plan** (2026-11-09), paquete `internal/plan`:
  - Un paso tiene `ID`, `Title`, `Reason`, `Command` y `Root`. `Validate`
    rechaza pasos sin motivo: nada se aprueba sin explicación.
  - `Command` es una lista de argumentos y se ejecuta sin shell. Así no hay
    problemas de comillas ni inyección, y el comando mostrado es el ejecutado.
  - `Root` marca qué necesita privilegios; si se usa `sudo` lo decide quien
    ejecute, no el plan.
  - Un plan vacío es válido: significa "no hay nada que hacer".
  - Construir o mostrar un plan nunca ejecuta nada; ejecutar será otro paquete.
  - Se serializa a JSON (librería estándar) para poder guardarlo o revisarlo.
  - Un paso hace una sola cosa: ejecutar `Command` o escribir `File` (ruta
    absoluta, contenido y permisos). Escribir archivos así evita
    `sh -c "echo ... > archivo"`.
- **D12. Instalación de Docker** (2026-11-09), paquete `internal/docker`:
  - Sigue la guía oficial de Docker para Debian: repositorio de Docker
    (formato deb822) y paquetes `docker-ce`, `docker-ce-cli`, `containerd.io`,
    `docker-buildx-plugin` y `docker-compose-plugin`. No se usa `docker.io` de
    Debian porque no trae el plugin de Compose que documenta Docker.
  - Tampoco se usa el script `get.docker.com`: ejecuta código remoto sin que
    el usuario vea cada paso, lo contrario de la idea de CaratCore.
  - CaratCore no borra ni modifica nada existente. Se detiene con un mensaje
    claro si: la plataforma no es Debian 13 amd64; hay paquetes en conflicto
    (`docker.io`, `containerd`, `runc`...); o Docker está instalado sin Compose.
  - Si Docker y Compose ya están, el plan sale vacío.
  - `Detect` (lee el sistema) está separado de `InstallPlan` (función pura),
    para poder probar el plan sin un Debian real.
  - Al principio solo `--dry-run`; la ejecución llegó con D18.
- **D13. `internal/osrelease`** (2026-11-09): la lectura de `/etc/os-release`
  sale de `doctor` a su propio paquete porque ahora la usan `doctor` y
  `docker`.
- **D14. Finales de línea LF** (2026-11-09): `.gitattributes` fuerza LF
  también en Windows, porque CaratCore genera archivos para Linux.
- **D15. Distribución como binario** (2026-11-09): el usuario no necesita Go.
  - Al subir una etiqueta `v*`, GitHub Actions (`release.yml`) pasa vet y
    tests, compila `caratcore-linux-amd64` estático (`CGO_ENABLED=0`) y crea
    una release en **borrador** con el binario y `SHA256SUMS`. Una persona la
    revisa y la publica.
  - La versión se graba al compilar (`-X main.version=...`); `caratcore
    version` la muestra. Compilaciones locales dicen `dev`.
  - Workflow propio en vez de GoReleaser: para un solo binario basta y hay una
    herramienta menos que entender.
  - Sin instalador `curl | sh`, por coherencia con la idea del proyecto.
  - No se publica una versión sin haberla probado en Franc o en una VM
    Debian limpia (ver D19 y D28).
- **D16. CI** (2026-11-09): `ci.yml` pasa `gofmt`, `go vet` y `go test` en cada
  push a `main` y en cada pull request.
- **D17. Docker siempre con sudo, nunca con el grupo `docker`** (2026-11-09).
  Estar en el grupo `docker` equivale a ser root sin contraseña ni registro.
  CaratCore no añade usuarios al grupo y usará `sudo docker ...`. `doctor`
  informa (INFO, no WARN) cuando el usuario no puede usar Docker sin sudo.
- **D18. Ejecutor del plan** (2026-11-09), paquete `internal/runner`:
  - La CLI muestra el plan y pide escribir `yes` (o `y`); cualquier otra
    respuesta cancela. Pedir confirmación es cosa de la CLI, no del núcleo.
    `--json` solo con `--dry-run`.
  - Si CaratCore no es root, los pasos `Root` se ejecutan con `sudo --`. Antes
    del primer paso se ejecuta `sudo -v`: si el usuario no puede usar sudo,
    falla sin haber tocado nada. La contraseña la pide sudo; CaratCore nunca
    la ve ni la guarda.
  - Los pasos `File` escriben el contenido en un temporal (permisos 0600) y lo
    copian con `install -m MODO -- temporal destino` (con sudo si hace falta):
    sin shell y con el comando real visible.
  - Cada comando se muestra (`$ ...`) antes de ejecutarse y su salida va
    directa a la terminal.
  - Para en el primer fallo y no deshace nada; informa qué pasos se
    completaron y cuáles no se ejecutaron. Los pasos de Docker se pueden
    repetir sin riesgo, así que basta con volver a lanzar el comando: se
    planifica de nuevo desde el estado actual. El rollback sigue pendiente.
  - Ctrl+C no mata al comando en curso (matar `apt`/`dpkg` con SIGKILL puede
    dejar el sistema a medias): el hijo recibe SIGINT del terminal, CaratCore
    sobrevive, no lanza más pasos e informa dónde se paró.
  - Sin opción `--yes` por ahora: no hace falta hasta que haya automatización.
- **D19. Pruebas reales en Franc, no en una VM** (2026-11-09). Franc es un
  portátil Debian 13 sin datos importantes y se puede reinstalar; es más
  práctico que mantener una VM. Sustituye a la regla anterior ("solo VM,
  jamás Franc"). Riesgo aceptado: un fallo puede cortar Tailscale o dejar
  `apt` a medias, así que las pruebas que cambian el sistema se hacen estando
  delante del portátil. Para probar la instalación, antes hay que desinstalar
  el Docker que ya tiene.
- **D20. Servicios de la v1: AdGuard Home y Uptime Kuma** (2026-11-09; antes
  P4). Solo dos: lo difícil de la v1 es la parte que despliega (compose,
  datos, puertos, comprobación, actualización), y con dos servicios se
  construye y se valida. AdGuard Home en lugar de Pi-hole: un solo contenedor
  oficial, DNS cifrado (DoH/DoT) incluido y buena interfaz web. Uptime Kuma
  avisa si un servicio cae sin necesitar HTTPS. Tailscale se detecta, no se
  instala.
- **D21. Despliegue con Docker Compose** (2026-11-09; antes P2). CaratCore
  genera y explica los archivos y ejecuta `sudo docker compose`; no hay motor
  de despliegue propio.
- **D22. Catálogo en JSON, no YAML** (2026-11-09; antes P3). Leer YAML exige
  una dependencia externa y la regla es usar solo la librería estándar
  (`encoding/json`). El catálogo va incrustado con `go:embed`. El
  `compose.yaml` se genera con `text/template`, porque escribir YAML no
  necesita ninguna librería; leerlo, sí.
- **D23. Despliegue de servicios** (2026-11-09), paquetes `internal/catalog`
  y `internal/service`:
  - Cada servicio vive en `/opt/caratcore/<id>/`: su `compose.yaml` y sus
    carpetas de datos (bind mounts, 0700 porque guardan hashes de
    contraseñas). Todo en un sitio que se puede leer y copiar.
  - Imágenes con versión fija, nunca `latest`; el catálogo lo valida. La misma
    versión de CaratCore despliega siempre lo mismo. Versiones comprobadas en
    las releases de GitHub (2026-11-09): AdGuard Home v0.107.80 y Uptime Kuma
    2.5.9 (`latest` de Uptime Kuma sigue apuntando a la 1.x).
  - Plan de 4 pasos, todos con sudo (D17): crear carpetas, escribir
    `compose.yaml`, `docker compose pull` y `docker compose up --detach`. El
    `pull` va aparte para que un fallo de red pare antes de arrancar nada.
    `restart: unless-stopped` arranca el servicio al encender la máquina.
  - Se niega si no hay Docker y Compose, si un puerto está ocupado por otro
    programa, o si existe un `compose.yaml` distinto del que generaría (lo
    editó el usuario u otra versión): no se toca. Si es idéntico, el plan se
    repite sin riesgo, así que relanzar tras un fallo funciona (como en D18).
  - AdGuard Home en red bridge, publicando solo 53/tcp, 53/udp y 3000. Sin
    DHCP ni DNS cifrado por ahora. Su asistente propone el puerto 80 para el
    panel; hay que dejar el 3000, y CaratCore lo indica en los pasos
    siguientes. No se preconfigura `AdGuardHome.yaml` porque obligaría a
    CaratCore a manejar la contraseña del administrador.
  - La lectura de puertos de `/proc/net` sale de `doctor` a `internal/ports`;
    los puertos que comprueba `doctor` salen del catálogo.
- **D24. `doctor` reconoce los servicios desplegados** (2026-11-09). Si existe
  `/opt/caratcore/<id>/compose.yaml`, los puertos ocupados de ese servicio
  salen como OK ("in use by services CaratCore deployed"); los demás puertos
  ocupados siguen siendo WARN. Limitación aceptada: `doctor` no confirma que
  quien escucha en el puerto sea el contenedor del servicio, porque
  preguntárselo a Docker exige sudo (D17) y `doctor` no lo usa. Si el
  contenedor está parado y otro programa ocupa el puerto, `doctor` no lo
  verá.
- **D25. Puertos publicados y cortafuegos: se documenta, no se limita**
  (2026-11-09). Docker desvía los puertos publicados en la tabla `nat` antes
  de la cadena `INPUT` que usa ufw, así que ufw no los bloquea. Se descartan
  dos alternativas: publicar en `127.0.0.1`, que rompe el acceso desde la red
  local y desde Tailscale, y publicar en la IP de Tailscale, que hace fallar
  el arranque del contenedor si Tailscale aún no está levantado. El objetivo
  de la v1 es un servidor doméstico detrás de un router, donde la exposición a
  la red local es la deseada. El README advierte de no abrir los puertos en el
  router ni desplegar en una máquina con IP pública (resolvedor DNS abierto)
  sin filtrar con `DOCKER-USER`.
- **D26. Clave GPG de Docker incrustada en el binario** (2026-11-09). En
  lugar de descargarla con curl en cada instalación (como la guía oficial),
  `internal/docker/docker.asc` va con `go:embed` y se escribe con un paso
  `File` (0644). Un test calcula su huella OpenPGP v4 con la librería
  estándar y la compara con la de la guía de Docker
  (`9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88`, comprobada también
  con `gpg --show-keys`). Ventajas: la clave se revisa en el código, no
  depende de la red en el momento de instalar y sobra curl como
  prerrequisito. Coste aceptado: si Docker rota la clave, hace falta una
  versión nueva de CaratCore (como con las imágenes fijas de D23). El plan
  muestra la clave entera; el motivo del paso incluye la huella y el comando
  para comprobarla.
- **D27. `doctor` mira el grupo `sudo`** (2026-11-09). Antes solo veía que
  sudo existía. Ahora: OK si el proceso está en el grupo `sudo` (el que
  Debian autoriza por defecto en sudoers), WARN si no lo está o si sudo no
  está instalado (con los comandos para arreglarlo), e INFO si no se puede
  saber. No se ejecuta `sudo -n` porque `doctor` nunca ejecuta sudo. Es una
  pista, no una prueba: sudoers puede configurarse de otra forma; la prueba
  real sigue siendo `sudo -v` en el ejecutor (D18). Se usan los grupos del
  proceso, que son los de la sesión, como los ve sudo: tras `adduser` hay que
  volver a entrar. CaratCore no instala sudo: hacerlo ya exige ser root y
  cambiar quién tiene privilegios es decisión del usuario; el README explica
  cómo.
- **D28. La v0.1.0 se prueba en una VM Debian 13 limpia** (2026-11-09).
  Complementa D19: Franc ya tiene Docker y servicios, así que no sirve para
  probar una instalación desde cero. La lista está en
  `docs/prueba-v0.1.0.md`: se etiqueta primero, se prueba en la VM el binario
  del borrador que crea `release.yml` (así se prueba exactamente lo que se
  publicará), y solo si pasa se publica y se repite la descarga del README.
  Si falla, se borran el borrador y la etiqueta, y nadie lo ha visto. No hace falta un
  pre-instalador: el binario es estático (sin Go ni librerías), y lo único que
  necesita es root o sudo.
- **D29. Alcance de la v0.2** (2026-11-10; antes P6 y P7): primero
  `uninstall` (D30), después el modo guiado (D31), en commits separados.
  Actualizar servicios (cambiar la versión de la imagen) queda para después.
  La prueba en VM (D28) se amplía con `uninstall`, `--purge` y un
  `compose.yaml` editado.
- **D30. `uninstall <servicio>`** (2026-11-10; antes P7). Hoy se puede
  desplegar pero no quitar nada; es el hueco funcional más grande.
  - Plan de un paso: `sudo docker compose down` en `/opt/caratcore/<id>/`.
    Si la carpeta no existe, el plan sale vacío; repetirlo es seguro.
  - `--purge` borra además `/opt/caratcore/<id>/` (datos incluidos) con
    `rm -r --`, sin shell. La ruta sale siempre del catálogo, nunca del
    usuario. Como no se puede deshacer, la confirmación exige escribir el id
    del servicio, no `yes`.
  - `down` funciona aunque el `compose.yaml` esté editado: para lo que dice
    el archivo y no modifica nada. `--purge` se niega en ese caso, por
    coherencia con D23 ("no se toca").
  - No borra imágenes (el mensaje final indica `sudo docker image rm <imagen>`;
    `prune` no sirve porque la imagen tiene etiqueta) ni
    desinstala Docker.
  - El motivo del paso de AdGuard Home avisa: si el router o Tailscale lo
    usan como DNS, la red se queda sin resolver nombres.
- **D31. Modo guiado** (2026-11-10; antes P6, idea de Bryan: que los pasos
  no pasen de golpe sino "como en los juegos", explicando poco a poco).
  - Solo con `--guided`. Nada de "por defecto la primera vez": exigiría
    estado persistente, que sigue pendiente.
  - La aprobación no cambia (D18): se muestra el plan entero y se pide `yes`.
    `--guided` solo marca el ritmo: antes de cada paso muestra `[3/7]`,
    título, porqué, `Learn` y comando, y espera Enter (`q` para, con el mismo
    informe que un fallo). Un Enter por paso como única aprobación sería un
    consentimiento débil (basta dejar la tecla pulsada). Sin "saltar paso":
    los pasos dependen unos de otros.
  - `Learn`: campo opcional del paso con una explicación más larga (keyring,
    deb822, Compose, bind mounts...). `Validate` no lo exige; se escribe
    solo donde aporta.
  - Colores y marcas (✓ / ✗) y un resumen final en ambos modos, con ANSI de
    la librería estándar, desactivados si la salida no es una terminal o si
    existe `NO_COLOR`. Es lo primero que se recorta si la v0.2 se alarga.
  - Sin librerías de interfaz de terminal: no hacen falta para esperar un
    Enter y pintar colores.
  - Implementación: `runner` gana un gancho opcional `Before`; esperar
    teclas es cosa de la CLI (`guided.go`), como confirmar (D18). Ctrl+C
    durante la espera para sin necesidad de Enter, y el runner vuelve a
    mirar el contexto tras el gancho. `StepError.NotStarted` distingue "no
    empezó" (Ctrl+C, `q`) de "falló", y el informe ya no da por ejecutado un
    paso que no empezó (antes pasaba con Ctrl+C). `--guided --dry-run` se
    rechaza: el modo guiado marca el ritmo de una ejecución real.
  - Colores: paquete `internal/term` (una responsabilidad: estilo de
    terminal). Su valor cero no pinta, así que tests y salida redirigida
    quedan en texto plano. Se apagan si la salida no es una terminal, si
    `NO_COLOR` no está vacío (como dice no-color.org) o si `TERM=dumb`. El
    plan (`Render`) y `doctor` siguen sin color: es lo que se revisa o se
    copia. Las marcas ✓ / ✗ salen siempre, con o sin color.
  - Resumen final: título del plan y pasos completados; al desplegar un
    servicio, además, dónde está y cómo quitarlo.
  - Prueba en la VM (2026-11-11, `docs/prueba-v0.2.0.md`): con Ctrl+C el
    informe decía `context canceled` y `Steps 1-1`; ahora dice
    `interrupted (Ctrl+C)` y `Step 1 was`. Con `NO_COLOR`, la salida de
    `docker compose` sigue en color porque `sudo` limpia el entorno; se
    acepta: no es texto de CaratCore y evitarlo ataría el plan a la terminal.
- **D32. Alcance de la v0.3: mantener lo desplegado** (2026-11-12). Hoy no
  se puede actualizar un servicio, y un servidor sin actualizar acaba siendo
  inseguro. Bloques, en orden y en commits separados:
  1. **Archivo de estado por servicio** (resuelve la deuda de D23): al
     desplegar se escribe `/opt/caratcore/<id>/caratcore.json` con la versión
     de CaratCore, la imagen y el hash del `compose.yaml` escrito. Así se
     distingue "editado a mano" de "escrito por otra versión". Va en la
     carpeta del servicio y no en `/var/lib/caratcore`: queda a la vista y
     `--purge` lo borra con el resto.
     Hecho (2026-11-12): paso `write-state` justo después de
     `write-compose`. Un `compose.yaml` de otra versión sin tocar hace que
     `install` se niegue nombrando versión e imagen (hasta que exista
     `update`) y `--purge` lo permite. Sin `caratcore.json`, o con uno
     ilegible, un archivo distinto se trata como editado. Lo desplegado con
     la v0.1 o la v0.2 coincide con lo que escribe esta versión (el
     catálogo no ha cambiado), así que repetir `install` escribe el estado
     sin tocar nada más: no hace falta migración.
  2. **Script de prueba en la VM**: los casos no interactivos dicen OK o
     FALLO solos; Bryan hace solo lo que necesita una persona (Ctrl+C,
     colores, navegador). Hecho (2026-11-12): `scripts/vm-test.sh`,
     en Bash (se ejecuta en la VM, no se distribuye), con los manuales en
     `docs/prueba-v0.3.0.md`.
  3. **`update <servicio>`**: escribir el `compose.yaml` con la imagen nueva,
     `pull` y `up -d`. Se actualiza **solo a la imagen del catálogo** del
     binario, no a la última de Docker Hub: versiones fijas y probadas, como
     hasta ahora; para tener imágenes nuevas se actualiza CaratCore. Si el
     `compose.yaml` fue editado a mano, se niega y explica por qué (D23).
     Antes de cambiar la imagen, el plan copia los datos del servicio a una
     carpeta `.bak` con fecha: una imagen nueva puede migrarlos sin vuelta
     atrás (Uptime Kuma 1→2). No son backups completos. Hecho (2026-11-12): D33.
  4. **`list`** (opcional): servicios instalados, su imagen y si están al
     día. Es lo primero que se recorta si la v0.3 se alarga. Hecho
     (2026-11-12): D34.

  Orden cambiado (2026-11-12): `update` y `list` antes que el script, para
  que el script pruebe la versión entera.

  Fuera de la v0.3: Homepage (v0.4, depende de poder actualizar), Windows
  (P9) e idioma (P8). Aparte del código, y sin esperar a la v0.3: enseñar la
  v0.2 en r/selfhosted o r/homelab y apuntar qué preguntan y qué piden
  ([vision.md](vision.md)).
- **D33. `update <servicio>`** (2026-11-12, bloque 3 de D32).
  - Solo actúa si el `compose.yaml` es de otra versión de CaratCore y nadie
    lo ha tocado (D32). Si está editado, se niega: actualizar borraría los
    cambios. Si no está instalado, remite a `install`.
  - Plan de 6 pasos, pensado para que un fallo deje el servicio lo mejor
    posible: `docker pull` de la imagen nueva (un fallo de red para antes de
    tocar nada, con el servicio aún en marcha); `docker compose stop` (una
    copia de una base de datos en uso puede salir rota); `cp -a` de la
    carpeta del servicio; escribir `compose.yaml` y `caratcore.json`; y
    `up --detach`, que recrea el contenedor con la imagen nueva.
  - La copia va **al lado** de la carpeta, no dentro:
    `/opt/caratcore/<id>.bak-AAAA-MM-DD-HHMMSS`. Dentro se copiaría a sí
    misma. Copia la carpeta entera (`compose.yaml`, estado y datos), así que
    volver atrás es cambiar una carpeta por otra; los pasos siguientes dan
    los comandos exactos. No es un backup (mismo disco, ver P11, v0.5).
  - `uninstall --purge` no borra las copias (solo borra `/opt/caratcore/<id>/`,
    D30), pero las nombra en los pasos siguientes para que nadie crea que no
    queda nada. `Detect` las encuentra con `filepath.Glob`.
  - Si el `compose.yaml` ya es el de esta versión, el plan solo contiene
    `up --detach` (y `write-state` si falta el estado). Es lo que queda si
    una actualización se cortó tras escribir el archivo; si no, no hace nada.
    Repetir `update` es seguro, como en D18.
  - Al actualizar AdGuard Home, el paso `stop` avisa de que la red se queda
    sin DNS hasta el último paso (el mismo aviso del catálogo que en D30).
  - `update docker` se niega: Docker viene de su repositorio apt y se
    actualiza con `apt upgrade`.
  - Limitación conocida: `update` arranca el servicio aunque se hubiera
    parado con `uninstall` sin `--purge` (CaratCore no consulta a Docker qué
    contenedores hay, para no necesitar sudo al planificar). Si una versión
    nueva añade un puerto ocupado, el fallo lo dará `up`, no el plan.
- **D34. `list`** (2026-11-12, bloque 4 de D32). Un estado por servicio del
  catálogo: `not installed`, `up to date`, `update available` o `edited`,
  con el detalle y, si hay algo que hacer, el comando. Además, las copias de
  `update`. Solo lee archivos de `/opt/caratcore`: no pregunta a Docker si
  el contenedor está en marcha, porque exigiría sudo (D17), y la salida lo
  dice. La lógica (`StatusOf`) está en `service`; la tabla
  (`text/tabwriter`), en la CLI.

## Propuestas (por confirmar)
- **P1. Usuario objetivo:** entusiasta principiante y curioso, no un usuario sin
  ninguna base técnica.
- **P5. Ejecutable de terminal**, no un servicio en segundo plano.
- **P8. Idioma según el host** (2026-11-10, idea de Bryan). Posible con la
  librería estándar (leer `LC_ALL`, `LC_MESSAGES` y `LANG`), pero duplica
  todos los textos (motivos, `Learn`, errores, `doctor`, catálogo, README) y
  cada paso nuevo exigiría dos. No antes de cerrar la v0.2. Si se hace,
  empezar por los `Learn` en español, con un test que exija los dos idiomas.
- **P9. Windows** (2026-11-11, Bryan: "tendrá que servir también en
  Windows"). D10 sigue vigente para la v1. Hay que concretar qué significa:
  (a) Windows como **servidor** (servicios en Docker Desktop/WSL2: otro
  instalador, red NAT, el equipo se suspende, puerto 53 ocupado por
  Windows), o (b) Windows como **equipo desde el que se maneja** un servidor
  Linux (por SSH). (b) encaja mejor con un homelab y con el plan como dato
  (D2); (a) es otro producto.
  - Caso de Bryan: un portátil viejo con Windows que se quiere usar de
    servidor. Ahí Windows estorba: Docker Desktop pide virtualización y RAM
    que un equipo viejo no tiene, y Windows se reinicia, se suspende y exige
    iniciar sesión. Propuesta: (a) descartada; en su lugar, una guía
    "convierte tu portátil viejo en servidor Debian" (USB, instalación sin
    escritorio, SSH), ampliando la del README. (b), como idea fuera de la v1.
- **P10. Manejar el servidor sin vivir en la terminal** (2026-11-11, idea
  de Bryan: administrarlo desde el PC con Windows o desde el móvil, con algo
  más visual que comandos). Tres piezas, en este orden:
  1. **Guía** (solo documentación, ya): Debian sin escritorio en un equipo
     viejo; entrar por SSH desde Windows (viene de serie) y desde el móvil
     (app de SSH); fuera de casa, con Tailscale.
  2. **Homepage** (tras actualizar servicios; ya era candidato de la v1.1):
     un panel con enlaces a los servicios, que ya tienen su propia web.
  3. **Panel web propio de CaratCore** (v2): otra interfaz para el mismo
     plan (D2; el plan ya sale en JSON). Antes de escribir código, una
     decisión de seguridad: sería un servicio siempre en marcha capaz de
     ejecutar como root, así que exige autenticación, HTTPS y sesiones, y
     choca con P5.
  Visión de producto relacionada: [vision.md](vision.md).
- **P11. Hoja de ruta** (2026-11-12, propuesta de Claude). Una versión, un
  tema; cada una se prueba en la VM y en Franc antes de publicarla (D28).
  Tras la v0.3, enseñarla en r/selfhosted: lo que pida la gente puede
  cambiar el orden a partir de la v0.5.
  | Versión | Tema | Contenido |
  |---|---|---|
  | v0.3 | Mantener | estado por servicio, `update` con copia de datos, `list`, script de prueba (D32) |
  | v0.4 | Acceso fácil | Homepage; guía "portátil viejo → Debian + SSH desde Windows y el móvil" (P10.1) |
  | v0.5 | Copias de seguridad | `backup` y `restore` de un servicio |
  | v0.6 | HTTPS | `tailscale serve`; Vaultwarden como primer servicio que lo necesita |
  | v0.7 | Más alcance | binarios arm64, experimento de Windows con WSL (P9), Jellyfin o Nextcloud si se piden |
  | v1.0 | Estable | todo probado en hardware real, `.deb` con `apt upgrade`, releases firmadas |
  | v2 | Panel web | ver y aprobar el mismo plan desde el navegador (P10.3) |

## Pendientes
- Hecho (2026-11-12): **v0.3.0 publicada** tras probar el binario del
  borrador en la VM `franc1` (`docs/prueba-v0.3.0.md`): script 31 OK, 0
  FALLO, y los casos manuales y el ensayo de adopción, bien. Un defecto de
  textos (`Learn` en `update`) se corrigió antes de etiquetar. La prueba en
  Franc (F) se hace **después** de publicar, por decisión de Bryan: el
  riesgo es bajo (si algo no cuadra, CaratCore se niega a tocarlo) y lo que
  encuentre irá a una v0.3.1. Hecho el mismo día: F pasó sin defectos.
- Hecho (2026-11-11): **v0.2.0 publicada** tras probar el borrador en la VM
  `franc1` (`docs/prueba-v0.2.0.md`): 15 de 16 casos (M5, opcional, sin
  hacer). Los dos textos corregidos (D31) obligaron a rehacer el borrador y
  la etiqueta; con el binario nuevo se repitieron el checksum, M3 y M4.
- Hecho (2026-11-10): **v0.1.0 publicada** tras probar el binario del
  borrador en una VM Debian 13 limpia (Hyper-V, `docs/prueba-v0.1.0.md`).
  Pasaron: checksum y versión; `doctor` en limpio (Privileges OK por grupo
  `sudo`, D27); `install docker` completo, clave escrita con la huella
  esperada (D26), `hello-world`, plan vacío al repetir; `install
  adguard-home` con asistente en el 3000 y DNS respondiendo; repetir el
  despliegue sin error de puertos; servicios activos tras reiniciar;
  `doctor` con los puertos del servicio como OK (D24); descarga pública con
  `wget` y `sha256sum -c`. No probado en la VM: los casos sin sudo (C y D;
  el instalador de Debian ya dio sudo al usuario), Uptime Kuma (sí probado
  antes en Franc) y los casos de fallo opcionales (G).
- Estado persistente (archivo o SQLite), rollback y actualizaciones.
- AdGuard Home como DNS de Tailscale: riesgo de perder internet si el servidor
  se apaga; necesita DNS de respaldo. **Plan (2026-11-12, Bryan), por hacer en
  Franc:** consola de Tailscale → DNS → nameserver personalizado con la IP
  `100.x` de Franc y "Override DNS servers"; AdGuard escuchando en todas las
  interfaces. El respaldo es manual: si Franc se apaga, se desconecta
  Tailscale en el dispositivo. Sin segundo DNS en Tailscale (los clientes
  usarían uno u otro al azar y se colarían anuncios). En Franc,
  `tailscale set --accept-dns=false`, para que no dependa de su propio
  AdGuard. Antes: IP fija de Franc en el router y `HandleLidSwitch=ignore`
  para que no se suspenda al cerrar la tapa. Idea: en red bridge, AdGuard ve
  todas las consultas con la IP de Docker, no la de cada dispositivo.
- Umbrales de RAM y disco de `doctor`: son estimaciones, hay que medirlas con
  los servicios reales.
- Probar los casos de fallo de `doctor` en una máquina real.
- Hecho (2026-11-09): AdGuard Home y Uptime Kuma desplegados en Franc con
  `install <servicio>`; las etiquetas de imagen existen y funcionan.
- Actualizar servicios (cambiar la versión de la imagen): en curso, D32.
  Desinstalar: D30.
- Probar en Franc: la detección de conflictos, un fallo a mitad y Ctrl+C
  durante `apt`. Hecho (2026-11-09): instalación real completa con
  `install docker`; con el usuario fuera del grupo `docker`, `doctor` muestra
  el Engine como INFO y `sudo docker ps` funciona.

## Ideas fuera de la v1
Interfaz web, otras distribuciones, GPU y recomendador avanzado, backups,
monetización (no antes de tener usuarios), paquete `.deb` o repositorio apt
para actualizar con `apt upgrade`, binarios para arm64 (Raspberry Pi),
firma de releases.

Servicios candidatos para la v1.1, en orden: **Homepage** (panel con enlaces
a los servicios; sin base de datos, configurado con archivos que CaratCore
puede generar y explicar) y **Vaultwarden** (gestor de contraseñas; exige
antes HTTPS, probablemente con `tailscale serve`, y backups). Descartados por
ahora: Nextcloud e Immich (pesados, mucho mantenimiento) y Jellyfin (depende
demasiado del hardware).
