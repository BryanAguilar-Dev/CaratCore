# Prueba de la v0.2.0 en una VM Debian 13

Se sigue el mismo método que en la v0.1.0 (D28, `docs/prueba-v0.1.0.md`):
etiquetar, probar el binario del borrador en la VM y publicar solo si pasa.
Esta lista crece con cada parte de la v0.2 (D29).

## M. Modo guiado (D31)

Punto de partida: la instantánea "limpia" de la v0.1.0 (sin Docker), con el
binario v0.2.0 copiado (apartados A y B de la v0.1.0).

- [x] (obligatorio) `install docker --guided --dry-run`: se niega.
- [x] (obligatorio) `install docker --guided`: muestra el plan entero y pide
  `yes`. Después, en cada paso se ven `[n/7]`, `Why`, `Learn` (en los que lo
  tienen) y `Does`, y solo continúa con Enter. Escribir otra cosa vuelve a
  preguntar.
- [x] (obligatorio) En el paso 4, escribir `q`: informa de que los pasos 1-3
  se completaron y que los pasos 4-7 no se ejecutaron. Relanzar `install
  docker --guided` y terminar: `hello-world` funciona.
- [x] (obligatorio) `install uptime-kuma --guided` hasta el paso 2 y pulsar
  Ctrl+C mientras espera: para al momento, sin pulsar Enter, e informa
  igual que con `q`.
- [ ] `install docker` sin `--guided` en otra instantánea limpia: igual que
  en la v0.1.0, sin pausas.
- [x] (obligatorio) Colores: títulos en negrita, `✓ done` en verde, el
  resumen final en verde con "all N step(s) completed", y al desplegar un
  servicio, la línea con su carpeta y `caratcore uninstall`.
- [x] (obligatorio) `NO_COLOR=1 ./caratcore-linux-amd64 uninstall uptime-kuma`
  y `... | cat`: sin códigos raros como `^[[1m`; las marcas ✓ / ✗ siguen.

## U. Desinstalar (D30)

Punto de partida: VM con Docker y los dos servicios desplegados (sirve una
VM con los servicios que desplegó la v0.1.0: el `compose.yaml` no ha
cambiado). Haz una instantánea antes.

- [x] (obligatorio) `uninstall uptime-kuma --dry-run`: un paso (`down`) y el
  motivo dice que los datos se quedan.
- [x] (obligatorio) `uninstall uptime-kuma`, `yes`. El panel del 3001 deja de
  responder; `/opt/caratcore/uptime-kuma/` sigue ahí. El mensaje final indica
  el comando `sudo docker image rm ...`.
- [x] (obligatorio) Repetir `uninstall uptime-kuma`: el plan vuelve a tener el
  paso `down` y aplicarlo no da error.
- [x] (obligatorio) `install uptime-kuma`: arranca con el usuario que ya
  existía (los datos se reutilizan).
- [x] (obligatorio) `uninstall adguard-home --purge`: el motivo del primer paso
  avisa del DNS; responder `yes` **cancela**; repetir y escribir
  `adguard-home` lo ejecuta. La carpeta desaparece.
- [x] (obligatorio) Repetir `uninstall adguard-home --purge`: "Nothing to do".
- [x] (obligatorio) `doctor` tras la purga: los puertos 53 y 3000 libres.
- [x] (obligatorio) Editar `/opt/caratcore/uptime-kuma/compose.yaml` (añadir
  un comentario): `uninstall uptime-kuma --purge` se niega sin tocar nada;
  `uninstall uptime-kuma` sin `--purge` sí funciona.
- [x] `uninstall docker`: se niega y remite a la guía de Docker.

## Defectos encontrados (2026-11-11)

La prueba se hace en la VM `franc1` (el portátil es `franc`).

- M4: con Ctrl+C el informe dice `not started: context canceled` (error
  interno de Go, no un mensaje para el usuario) y `Steps 1-1 were
  completed` cuando solo hay un paso. Funciona bien; son textos.
  **Corregido**: `interrupted (Ctrl+C)` y `Step 1 was completed`. Tras
  rehacer el borrador se repiten el checksum, M3 (`q`) y M4 (Ctrl+C).
- M7: con `NO_COLOR=1`, CaratCore no pinta, pero las líneas de `docker
  compose` (Pulled, Created, Started) salen en verde: `sudo` limpia el
  entorno y `NO_COLOR` no le llega. No es de CaratCore; se podría pasar
  `--ansi never` a Compose cuando el color está apagado. Menor.
  **Se deja así** (decisión de Bryan): es salida de Compose, y evitarlo
  haría que el plan dependiera de la terminal (D2).
- M5 (opcional) no se hizo.
