#!/usr/bin/env python3

import subprocess

def git_config(key, value):
    subprocess.run(
        ["git", "config", key, value],
        check=True
    )

print("Configurando Git para usar merge en lugar de rebase...")

git_config("pull.rebase", "false")
git_config("pull.ff", "false")
git_config("merge.ff", "false")

print("✓ Configuración aplicada:")
print("  pull.rebase = false")
print("  pull.ff     = false")
print("  merge.ff    = false")
