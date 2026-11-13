# Disclaimer

[Leer en español](#aviso-legal)

**augmented brain is beta software.** It is still being tested and may have bugs. A
person uses it every day on Ubuntu and has tried it on Windows 11; on macOS it is tested
in CI only. Back up anything important before you rely on it.

- **What you have keeps working, and from 1.0 on that is a promise.** When a version
  changes how notes or settings are kept, it brings the step that moves yours forward,
  and runs it by itself after a full copy. A command, setting or tool that changes name
  keeps working under the old one and says what replaced it. The betas (0.x) already
  work this way, but it is not a promise yet. What every 1.x keeps is in
  [What stays stable](https://icalvete.github.io/augmented-brain/guide/compatibility.html).
  What it still cannot do is in
  [Known limitations](https://icalvete.github.io/augmented-brain/guide/limitations.html).
- **No warranty.** augmented brain is provided "as is", without warranty of any kind,
  as stated in the [MIT license](LICENSE). The authors are not liable for any damage or
  loss arising from its use.
- **You are responsible for what your assistant does.** augmented brain helps an AI
  assistant work in a certain way, but the assistant runs commands and changes files on
  your machine and on any system it can reach. Review what it does, especially on
  production systems.
- **The guard is a safety net, not a security boundary.** It reads commands as text
  and catches common mistakes (pushing, running as root, writing secrets, touching
  production). It can be bypassed, and it does not replace permissions, backups or
  your own review.
- **Your notes and your privacy.** augmented brain does not send your notes anywhere,
  except to your own devices if you open the panel from them (`panel remote`, through
  your Tailscale network). Your assistant reads them to do its work, like any other file it opens, so their
  content travels with your conversations to the provider of the assistant you use.
  Do not keep in them anything you would not share with that provider. If you set up a
  backup, the notes also go to the git repository you choose; keep it private.
- **Not affiliated.** augmented brain is an independent open-source project. It is not
  affiliated with, endorsed by or sponsored by Anthropic. "Claude" and "Claude Code" are
  trademarks of Anthropic, PBC; other names are trademarks of their owners.

---

## Aviso legal

**augmented brain es software en fase beta.** Todavía está en pruebas y puede tener
errores. Una persona lo usa a diario en Ubuntu y lo ha probado en Windows 11; en macOS
solo se prueba en la CI. Haz copia de lo importante antes de confiar en él.

- **Lo que tienes sigue funcionando, y desde la 1.0 es una promesa.** Cuando una versión
  cambia cómo se guardan las notas o la configuración, trae el paso que pone al día las
  tuyas, y lo da sola después de una copia completa. Un comando, un ajuste o una
  herramienta que cambia de nombre sigue funcionando con el viejo y dice cuál lo
  sustituye. Las betas (0.x) ya funcionan así, pero todavía no es una promesa. Lo que
  mantiene toda 1.x está en
  [Lo que no cambia](https://icalvete.github.io/augmented-brain/guide/compatibility.es.html).
  Lo que todavía no sabe hacer está en
  [Limitaciones conocidas](https://icalvete.github.io/augmented-brain/guide/limitations.es.html).
- **Sin garantía.** augmented brain se ofrece "tal cual", sin garantía de ningún tipo,
  como dice la [licencia MIT](LICENSE). Los autores no responden de ningún daño o pérdida
  que resulte de su uso.
- **Eres responsable de lo que hace tu asistente.** augmented brain ayuda a un asistente
  de IA a trabajar de cierta forma, pero el asistente ejecuta comandos y cambia ficheros
  en tu máquina y en cualquier sistema al que llegue. Revisa lo que hace, sobre todo en
  sistemas de producción.
- **El guardia es una red contra errores, no una barrera de seguridad.** Lee los
  comandos como texto y detiene errores habituales (subir código, actuar como root,
  escribir secretos, tocar producción). Se puede eludir, y no sustituye a los permisos,
  las copias de seguridad ni tu propia revisión.
- **Tus notas y tu privacidad.** augmented brain no envía tus notas a ningún sitio,
  salvo a tus propios dispositivos si abres el panel desde ellos (`panel remote`, por tu
  red de Tailscale). Tu asistente las lee para trabajar, como cualquier otro fichero que abre, así que su
  contenido viaja con tus conversaciones al proveedor del asistente que uses. No guardes
  en ellas nada que no compartirías con ese proveedor. Si configuras una copia de
  seguridad, las notas van además al repositorio git que elijas; mantenlo privado.
- **Sin relación con terceros.** augmented brain es un proyecto independiente de código
  abierto. No está afiliado, respaldado ni patrocinado por Anthropic. "Claude" y "Claude
  Code" son marcas de Anthropic, PBC; los demás nombres son marcas de sus dueños.
