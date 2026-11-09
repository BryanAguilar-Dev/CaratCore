# Prueba de la v0.1.0 en una VM Debian 13

Lista para probar CaratCore desde un Debian recién instalado antes de publicar
la primera versión. Marca cada punto; si algo falla, apunta qué salió y no
etiquetes. Los puntos **(obligatorio)** deciden si se publica; los
**(opcional)** son casos de fallo que conviene probar pero no bloquean.

## A. Preparar la VM

- Imagen: Debian 13 netinst amd64 (debian.org).
- 2 GB de RAM y 20 GB de disco como mínimo.
- Red en modo **puente** (bridged), para que la VM tenga una IP de tu red y
  puedas abrir los paneles y probar el DNS desde Windows.
- En el instalador:
  - **Pon contraseña de root.** Así Debian no instala sudo y se prueba ese
    caso, que es el más difícil para un usuario nuevo.
  - Crea un usuario normal (en esta guía, `bryan`).
  - En "Selección de programas" marca solo **servidor SSH** y **utilidades
    estándar del sistema** (sin escritorio).
- Al terminar, apunta la IP de la VM (`ip -brief addr`) y **haz una
  instantánea** ("limpia"). Te permite volver atrás para los casos opcionales.

## B. Compilar y copiar el binario

En Windows, desde la carpeta del proyecto (PowerShell). Son las mismas
opciones que usa `release.yml`:

```
$env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -trimpath -ldflags "-s -w -X main.version=v0.1.0-test" -o caratcore-linux-amd64 ./cmd/caratcore
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
scp caratcore-linux-amd64 bryan@IP-DE-LA-VM:
```

En la VM (por SSH: `ssh bryan@IP-DE-LA-VM`):

```
chmod +x caratcore-linux-amd64
./caratcore-linux-amd64 version
```

- [ ] (obligatorio) Muestra `caratcore v0.1.0-test`.

El binario de Windows queda en la carpeta del proyecto, pero `.gitignore` lo
excluye de los commits.

## C. Sin sudo

- [ ] (obligatorio) `./caratcore-linux-amd64 doctor`: `Privileges` sale WARN,
  "sudo is not installed", y explica qué hacer.
- [ ] (obligatorio) `./caratcore-linux-amd64 install docker`, responde `yes`:
  falla con "cannot use sudo ... nothing was executed" y no cambia nada.

## D. Preparar sudo como dice el README

```
su -
apt install sudo
adduser bryan sudo
exit
```

- [ ] (obligatorio) Sin cerrar sesión, `doctor`: `Privileges` WARN, "not in
  the sudo group" (la sesión aún no tiene el grupo).
- [ ] (obligatorio) Sal de SSH, vuelve a entrar y `doctor`: `Privileges` OK.

## E. Instalar Docker

- [ ] (obligatorio) `install docker --dry-run`: 7 pasos, cada uno con su
  motivo. El paso "Write Docker's signing key" muestra la huella
  `9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88`.
- [ ] (obligatorio) `install docker`, responde `yes`: sudo pide la contraseña
  una vez y todos los pasos terminan bien.
- [ ] (obligatorio) La clave escrita es la incrustada:
  `sha256sum /etc/apt/keyrings/docker.asc` da
  `1500c1f56fa9e26b9b8f42452a553675796ade0807cdce11975eb98170b3a570`.
- [ ] (obligatorio) `sudo docker run --rm hello-world` funciona.
- [ ] (obligatorio) `doctor`: Docker CLI y Compose OK, Engine INFO (sin grupo
  `docker`, D17).
- [ ] (obligatorio) Repetir `install docker`: "Nothing to do".

## F. Desplegar los servicios

- [ ] (obligatorio) `install adguard-home`, `yes`. Abre
  `http://IP-DE-LA-VM:3000` desde Windows y completa el asistente **dejando
  el panel en el puerto 3000**.
- [ ] (obligatorio) El DNS responde desde Windows:
  `nslookup example.com IP-DE-LA-VM`.
- [ ] (obligatorio) `install uptime-kuma`, `yes`. Abre
  `http://IP-DE-LA-VM:3001` y crea el usuario.
- [ ] (obligatorio) `doctor`: `Ports` OK, "in use by services CaratCore
  deployed". **Copia la salida completa de `doctor`**: sustituirá el ejemplo
  del README.
- [ ] (obligatorio) Repetir `install adguard-home`: el plan se muestra y se
  puede aplicar sin error de "puerto ocupado".
- [ ] (obligatorio) `sudo reboot`; al volver, los dos paneles responden sin
  hacer nada (`restart: unless-stopped`).

## G. Casos de fallo (opcionales)

Vuelve a la instantánea "limpia" antes de cada uno.

- [ ] Como root: `su -`, y `install docker` sin sudo. Debe funcionar sin
  pedir contraseña.
- [ ] Conflicto: `sudo apt install docker.io` y luego `install docker`. Debe
  negarse y nombrar los paquetes, sin tocar nada.
- [ ] Ctrl+C durante "Install Docker Engine, CLI and plugins": `apt` termina o
  se interrumpe limpiamente, CaratCore informa qué pasos quedaron sin hacer y,
  al relanzar `install docker`, termina bien.
- [ ] Fallo a mitad: corta la red de la VM antes de responder `yes`. Sin red,
  `apt-get update` solo avisa y sale bien, así que lo esperable es que pare
  en la instalación de los paquetes ("Unable to locate package"), informe de
  qué pasos se hicieron y no siga. Con la red de vuelta, relanzar termina
  bien.

## H. Publicar

Si todos los obligatorios pasan:

1. Sustituir el ejemplo de `doctor` del README por la salida real (punto F) y
   apuntar el resultado de la prueba en `docs/decisions.md`.
2. Etiquetar y subir (desde Windows):
   ```
   git tag -a v0.1.0 -m "CaratCore v0.1.0"
   git push origin v0.1.0
   ```
3. En GitHub, pestaña *Actions*: esperar a que termine "Release". En
   *Releases* aparece el borrador con `caratcore-linux-amd64` y `SHA256SUMS`.
   Revisar las notas y pulsar **Publish release**.
4. Comprobación final en la VM: seguir el paso 2 del README ("Download
   CaratCore and check it") con `v0.1.0`. `sha256sum -c` debe decir `OK` y
   `version` debe mostrar `caratcore v0.1.0`.
