# Security

*En español, más abajo.*

## Reporting a vulnerability

If you find a security problem in augmented brain (the program, the installer, the
plugin or the panel), please do not open a public issue. Report it privately instead:

- On GitHub: **Security → Report a vulnerability**, at
  https://github.com/icalvete/augmented-brain/security/advisories/new

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

---

# Seguridad

## Avisar de una vulnerabilidad

Si encuentras un problema de seguridad en augmented brain (el programa, el instalador,
el plugin o el panel), no abras una incidencia pública. Avisa en privado:

- En GitHub: **Security → Report a vulnerability**, en
  https://github.com/icalvete/augmented-brain/security/advisories/new

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
