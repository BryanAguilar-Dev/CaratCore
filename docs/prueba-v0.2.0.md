# Prueba de la v0.2.0 en una VM Debian 13

Se sigue el mismo método que en la v0.1.0 (D28, `docs/prueba-v0.1.0.md`):
etiquetar, probar el binario del borrador en la VM y publicar solo si pasa.
Esta lista crece con cada parte de la v0.2 (D29); el modo guiado se añadirá
cuando exista.

Punto de partida: VM con Docker y los dos servicios desplegados con la
v0.2.0 (apartados A, B, E y F de la v0.1.0). Haz una instantánea antes.

## U. Desinstalar (D30)

- [ ] (obligatorio) `uninstall uptime-kuma --dry-run`: un paso (`down`) y el
  motivo dice que los datos se quedan.
- [ ] (obligatorio) `uninstall uptime-kuma`, `yes`. El panel del 3001 deja de
  responder; `/opt/caratcore/uptime-kuma/` sigue ahí. El mensaje final indica
  el comando `sudo docker image rm ...`.
- [ ] (obligatorio) Repetir `uninstall uptime-kuma`: el plan vuelve a tener el
  paso `down` y aplicarlo no da error.
- [ ] (obligatorio) `install uptime-kuma`: arranca con el usuario que ya
  existía (los datos se reutilizan).
- [ ] (obligatorio) `uninstall adguard-home --purge`: el motivo del primer paso
  avisa del DNS; responder `yes` **cancela**; repetir y escribir
  `adguard-home` lo ejecuta. La carpeta desaparece.
- [ ] (obligatorio) Repetir `uninstall adguard-home --purge`: "Nothing to do".
- [ ] (obligatorio) `doctor` tras la purga: los puertos 53 y 3000 libres.
- [ ] (obligatorio) Editar `/opt/caratcore/uptime-kuma/compose.yaml` (añadir
  un comentario): `uninstall uptime-kuma --purge` se niega sin tocar nada;
  `uninstall uptime-kuma` sin `--purge` sí funciona.
- [ ] `uninstall docker`: se niega y remite a la guía de Docker.
