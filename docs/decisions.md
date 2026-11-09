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
  - No se publica una versión sin haberla probado en Franc (ver D19).
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

## Propuestas (por confirmar)
- **P1. Usuario objetivo:** entusiasta principiante y curioso, no un usuario sin
  ninguna base técnica.
- **P5. Ejecutable de terminal**, no un servicio en segundo plano.

## Pendientes
- Estado persistente (archivo o SQLite), rollback y actualizaciones.
- AdGuard Home como DNS de Tailscale: riesgo de perder internet si el servidor
  se apaga; necesita DNS de respaldo.
- `doctor` solo detecta que sudo existe; el ejecutor ya lo comprueba con
  `sudo -v` antes de ejecutar (D18).
- Umbrales de RAM y disco de `doctor`: son estimaciones, hay que medirlas con
  los servicios reales.
- Probar los casos de fallo de `doctor` en una máquina real.
- Verificar la huella de la clave GPG de Docker además de descargarla por
  HTTPS (la guía oficial solo la descarga).
- Probar en Franc el despliegue real de AdGuard Home y Uptime Kuma, y
  confirmar que las etiquetas de imagen existen en Docker Hub.
- `doctor` marca WARN los puertos ocupados aunque los ocupe un servicio que
  desplegó CaratCore; debería reconocerlo.
- Actualizar servicios (cambiar la versión de la imagen) y desinstalarlos.
- Puertos publicados por Docker se saltan el cortafuegos (ufw/nftables): el
  DNS de AdGuard queda abierto a la red local. Documentarlo o limitarlo.
- Probar en Franc: la detección de conflictos, un fallo a mitad y Ctrl+C
  durante `apt`. Hecho (2026-11-09): instalación real completa con
  `install docker`; con el usuario fuera del grupo `docker`, `doctor` muestra
  el Engine como INFO y `sudo docker ps` funciona.
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
