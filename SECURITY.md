# Security

*En español, más abajo.*

## Reporting a vulnerability

If you find a security problem in augmented brain (the program, the installer, the
plugin or the panel), please do not open a public issue. Report it privately instead:

- On GitHub: **Security → Report a vulnerability**, at
  https://github.com/icalvete/augmented-brain/security/advisories/new

What counts, among others:

- **A way past the guard** that an assistant could take by mistake or be talked into:
  a command that pushes, runs as root, writes a secret or touches production without
  being stopped or asked about. (The guard is a safety net, not a wall, and
  [Known limitations](docs/guide/limitations.md#the-guard) lists what it cannot see;
  something on that list is not a vulnerability. When in doubt, report it privately.)
- **The panel** answering someone it should not, or showing or changing something
  without its key.
- **Your notes or secrets** reaching somewhere they should not: a secret value written
  to a note or a backup, a file readable by other users, the installer or an update
  running something it did not download and check.

A plain bug that only you can trigger, with no one gaining anything, is a public issue.

Say what you found, how to reproduce it and what someone could do with it. You will get
an answer within a week. Once it is fixed, the advisory is published with the release
that fixes it, and you are credited unless you prefer not to be.

## Which versions get fixes

Only the latest release. augmented brain is in beta: a fix comes out as a new release,
and `augmented-brain update` installs it.

## How it is watched

Every change, and every week, the CI looks for known vulnerabilities in Go's own library
and in every module the program uses (`govulncheck`, for Linux, macOS and Windows).
Dependabot proposes newer versions of the modules and of the CI's actions.

Every release is built by GitHub from a commit of `main` whose required checks all
passed, the macOS ones included (a pull request does not wait for them; a release does), with every file attested (`gh attestation verify`) and listed in `checksums.txt`;
the build is reproducible. How to check a download: [Installing](docs/guide/install.md#check-a-download).

---

# Seguridad

## Avisar de una vulnerabilidad

Si encuentras un problema de seguridad en augmented brain (el programa, el instalador,
el plugin o el panel), no abras una incidencia pública. Avisa en privado:

- En GitHub: **Security → Report a vulnerability**, en
  https://github.com/icalvete/augmented-brain/security/advisories/new

Lo que cuenta, entre otras cosas:

- **Una forma de saltarse el guardia** que un asistente podría tomar por error o porque
  le convenzan: un comando que sube código, actúa como root, escribe un secreto o toca
  producción sin que lo pare ni lo pregunte. (El guardia es una red, no un muro, y
  [Limitaciones conocidas](docs/guide/limitations.es.md#el-guardia) lista lo que no puede
  ver; lo que está en esa lista no es una vulnerabilidad. Si dudas, avisa en privado.)
- **El panel** contestando a quien no debe, o enseñando o cambiando algo sin su llave.
- **Tus notas o tus secretos** llegando adonde no deben: el valor de un secreto escrito
  en una nota o en una copia, un fichero que pueden leer otros usuarios, el instalador o
  una actualización ejecutando algo que no ha descargado y comprobado.

Un fallo normal que solo puedes provocar tú, sin que nadie gane nada, va a una incidencia
pública.

Cuenta qué has encontrado, cómo reproducirlo y qué podría hacer alguien con ello. Tendrás
respuesta en una semana como mucho. Cuando esté arreglado, el aviso se publica con la
versión que lo arregla, y se te nombra salvo que prefieras que no.

## Qué versiones reciben arreglos

Solo la última publicada. augmented brain está en beta: un arreglo sale en una versión
nueva, y `augmented-brain update` la instala.

## Cómo se vigila

Con cada cambio, y cada semana, la CI busca vulnerabilidades conocidas en la biblioteca
de Go y en cada módulo que usa el programa (`govulncheck`, para Linux, macOS y Windows).
Dependabot propone versiones nuevas de los módulos y de las acciones de la CI.

Cada versión la compila GitHub desde un commit de `main` que ha pasado todas las pruebas
obligatorias, también las de macOS (una pull request no los espera; una versión sí), con cada fichero atestado (`gh attestation verify`) y en `checksums.txt`; la
compilación es reproducible. Cómo comprobar una descarga: [Instalación](docs/guide/install.es.md#comprobar-una-descarga).
