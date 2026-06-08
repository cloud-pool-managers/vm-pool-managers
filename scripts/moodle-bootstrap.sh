#!/usr/bin/env bash
# Bootstrap du Moodle local de dev : active les Web Services, crée un service + token,
# et crée des cours/élèves/inscriptions de démo. Idempotent.
# Reporte MOODLE_URL / MOODLE_TOKEN dans le .env racine.
#
#   moodle/docker compose up -d   (Moodle doit répondre sur :8081)
#   scripts/moodle-bootstrap.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MOODLE_DIR="$ROOT/moodle"
MOODLE_URL="${MOODLE_URL:-http://localhost:8081}"

cd "$MOODLE_DIR"
echo "→ Vérification que Moodle répond sur $MOODLE_URL …"
if ! curl -s -o /dev/null --max-time 8 "$MOODLE_URL/login/index.php"; then
  echo "✗ Moodle ne répond pas. Lancer d'abord : (cd moodle && docker compose up -d) puis attendre l'install."
  exit 1
fi

echo "→ Exécution du bootstrap dans le conteneur (API Moodle)…"
docker compose cp bootstrap.php moodle:/tmp/cpm-bootstrap.php >/dev/null
OUT="$(docker compose exec -T moodle php /tmp/cpm-bootstrap.php)"
echo "$OUT"

if ! grep -q '^OK$' <<<"$OUT"; then
  echo "✗ Bootstrap incomplet (pas de ligne OK). Voir la sortie ci-dessus."
  exit 1
fi
TOKEN="$(grep '^TOKEN=' <<<"$OUT" | head -1 | cut -d= -f2)"
if [ -z "$TOKEN" ]; then echo "✗ Token introuvable dans la sortie."; exit 1; fi

# ── Reporter MOODLE_URL / MOODLE_TOKEN dans le .env racine (idempotent) ──
ENV="$ROOT/.env"
touch "$ENV"
upsert() { # clé valeur
  local k="$1" v="$2"
  if grep -qE "^${k}=" "$ENV"; then
    # remplace la valeur existante (compatible macOS/BSD sed)
    sed -i '' -E "s|^${k}=.*|${k}=${v}|" "$ENV" 2>/dev/null || sed -i -E "s|^${k}=.*|${k}=${v}|" "$ENV"
  else
    printf '\n%s=%s\n' "$k" "$v" >> "$ENV"
  fi
}
upsert "MOODLE_URL" "$MOODLE_URL"
upsert "MOODLE_TOKEN" "$TOKEN"

echo ""
echo "✓ Moodle prêt. MOODLE_URL et MOODLE_TOKEN écrits dans .env"
echo "  UI Moodle  : $MOODLE_URL  (admin / voir moodle/.env)"
echo "  Token (WS) : ${TOKEN:0:8}… (longueur ${#TOKEN})"
