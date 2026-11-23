# Git Flow — Guía de trabajo



## Estructura de ramas



```

main       ← producción, siempre estable

└── develop ← integración, aquí llegan todas las features

    └── feat/nombre-feature ← una rama por funcionalidad

```



| Rama | Propósito | ¿Se hace merge aquí? |

|------|-----------|----------------------|

| `main` | Código en producción | Solo desde `develop`, por el responsable del proyecto |

| `develop` | Integración continua | Sí, desde cada `feat/*` |

| `feat/nombre-feature` | Desarrollo de una funcionalidad | No recibe merges, solo se mergea hacia `develop` |



---



## 1. Setup inicial (obligatorio)



Antes de comenzar a trabajar, cada integrante debe ejecutar el script de configuración **una sola vez** por repositorio clonado:



```bash

python git/setup_git.py

```



Esto configura Git para que todos los pulls y merges generen un merge commit explícito, evitando rebase involuntarios y manteniendo un historial claro.



---



## 2. Mantenerse al día con `main`



Cada vez que haya cambios en `main` (por ejemplo, un hotfix o un release), **todos los integrantes deben incorporarlos a su rama local de `develop`**:



```bash

# 1. Actualizar main local

git checkout main

git pull origin main



# 2. Llevar esos cambios a develop

git checkout develop

git merge main

```



> Si hay conflictos, resuélvelos en `develop`, nunca en `main`.



---



## 3. Actualizar `develop` antes de crear o continuar una feature



Antes de empezar cualquier trabajo, sincroniza `develop` con el remoto para evitar conflictos posteriores:



```bash

git checkout develop

git pull origin develop

```



Como el setup tiene `pull.ff = false`, este pull siempre generará un merge commit si hay divergencia. Eso es intencional: el historial queda explícito.



---



## 4. Crear una feature



Siempre parte desde `develop` actualizado:



```bash

git checkout develop

git pull origin develop



git checkout -b feat/nombre-feature

```



Convención de nombres: `feat/` seguido de un nombre corto en kebab-case que describa la funcionalidad.



```

feat/login-usuario

feat/panel-administrador

feat/envio-correo-bienvenida

```



---



## 5. Trabajar en la feature



Haz commits atómicos y descriptivos mientras desarrollas:



```bash

git add .

git commit -m "feat: descripción corta de lo que hace el commit"

```



Si la feature toma varios días, mantén tu rama actualizada respecto a `develop`:



```bash

git checkout develop

git pull origin develop

git checkout feat/nombre-feature

git merge develop

```



Resuelve conflictos en tu rama `feat/*`, no en `develop`.



---



## 6. Mergear la feature a `develop`



Cuando la funcionalidad esté lista:



```bash

# 1. Asegúrate de que develop esté al día

git checkout develop

git pull origin develop



# 2. Mergea tu feature (nunca uses --ff)

git merge feat/nombre-feature



# 3. Sube develop al remoto

git push origin develop

```



> **Regla crítica:** Las ramas `feat/*` **nunca** se mergean directamente a `main`.



---



## 7. Eliminar la rama de feature



Una vez mergeada y subida, borra la rama para mantener el repositorio limpio:



```bash

# Eliminar local

git branch -d feat/nombre-feature



# Eliminar remoto

git push origin --delete feat/nombre-feature

```



---



## Flujo resumido



```

main

 │

 └──► develop  ◄──── git pull origin develop (antes de empezar)

          │

          └──► feat/nombre-feature

                    │  (desarrollar y commitear)

                    │

                    └──► merge a develop

                              │

                              └──► git push origin develop

```



---



## Reglas rápidas



- Siempre correr `python git/setup_git.py` al clonar el repo

- Siempre partir de `develop` actualizado al crear una feature

- Actualizar `develop` si hay cambios en `main`

- Mergear `feat/*` → `develop`

- ❌ Nunca mergear `feat/*` directamente a `main`

- ❌ Nunca trabajar directamente en `main` o `develop`
