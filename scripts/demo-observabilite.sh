#!/usr/bin/env bash
# Jeu de test pour le dashboard d'observabilité (Grafana « CloudPoolManager — Usage »).
#
# Insère des données de TEST clairement préfixées « demo-observ » :
#   - des VMs "actives" (table vm_instances)        -> compteur « VMs actives »
#   - des connexions GitHub (table git_hub_sessions) -> compteur « Connexions GitHub (24 h) »
# Toutes les lignes sont supprimables proprement (préfixe demo-observ).
#
# Usage :
#   scripts/demo-observabilite.sh demo     # montée progressive 0->5 VMs actives + sessions, puis nettoyage auto
#   scripts/demo-observabilite.sh up [N]   # insère N VMs actives (défaut 3) + 1 session GitHub, et laisse en place
#   scripts/demo-observabilite.sh down     # supprime TOUTES les données de test demo-observ
#   scripts/demo-observabilite.sh status   # compte les lignes de test présentes
#
# Le dashboard se rafraîchit toutes les 30 s (scrape Prometheus 30 s) -> compter ~30-60 s
# entre une action ici et son apparition dans Grafana.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
eval "$(grep -E '^POSTGRES_(USER|PASSWORD|DB|HOST|PORT)=' .env | sed 's/^/PG_/')"
PG_HOST="${PG_POSTGRES_HOST:-localhost}"; PG_PORT="${PG_POSTGRES_PORT:-5432}"
PREFIX="demo-observ"

psql_run() { PGPASSWORD="$PG_POSTGRES_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_POSTGRES_USER" -d "$PG_POSTGRES_DB" -X -q -t -A "$@"; }

add_vm() { # $1 = index
  local i="$1"
  psql_run -c "INSERT INTO vm_instances (id, name, ip, status, healthy, activity_status, registered_at, last_seen, raw_meta)
    VALUES ('${PREFIX}-vm-${i}', '${PREFIX}-vm-${i}', '10.255.0.${i}', 'ready', true, 'active', now(), now(),
            '{\"demo\":true,\"serverpool_id\":\"demo-pool\",\"user_id\":\"demo@local\"}')
    ON CONFLICT (id) DO UPDATE SET activity_status='active', last_seen=now();" >/dev/null
}
add_session() { # $1 = index
  local i="$1"
  psql_run -c "INSERT INTO git_hub_sessions (id, login, ssh_keys, created_at)
    VALUES ('${PREFIX}-gh-${i}', '${PREFIX}-student-${i}', '[]', now())
    ON CONFLICT (id) DO UPDATE SET created_at=now();" >/dev/null
}
clean() {
  psql_run -c "DELETE FROM vm_instances WHERE id LIKE '${PREFIX}-%';" >/dev/null
  psql_run -c "DELETE FROM git_hub_sessions WHERE id LIKE '${PREFIX}-%' OR login LIKE '${PREFIX}-%';" >/dev/null
}
status() {
  local v g
  v=$(psql_run -c "SELECT count(*) FROM vm_instances WHERE id LIKE '${PREFIX}-%';")
  g=$(psql_run -c "SELECT count(*) FROM git_hub_sessions WHERE id LIKE '${PREFIX}-%';")
  echo "VMs de test actives : $v | sessions GitHub de test : $g"
}

cmd="${1:-demo}"
case "$cmd" in
  up)
    n="${2:-3}"
    for i in $(seq 1 "$n"); do add_vm "$i"; done
    add_session 1
    echo "Inséré : $n VM(s) active(s) + 1 session GitHub (préfixe ${PREFIX})."
    status
    echo "→ Regarde Grafana (~30-60 s). Pour nettoyer : scripts/demo-observabilite.sh down"
    ;;
  down) clean; echo "Données de test supprimées."; status ;;
  status) status ;;
  demo)
    trap 'echo; echo "Nettoyage…"; clean; status; exit 0' INT TERM
    echo "Montée progressive — ouvre Grafana et regarde « VMs actives » grimper. Ctrl-C pour arrêter/nettoyer."
    for i in 1 2 3 4 5; do
      add_vm "$i"; add_session "$i"
      echo "  + VM active #$i (total test: $i) — attente 25 s…"
      sleep 25
    done
    echo "Pic atteint (5 VMs actives). Maintien 60 s puis nettoyage automatique…"
    sleep 60
    echo "Nettoyage…"; clean; status
    echo "Terminé — « VMs actives » doit redescendre à sa valeur réelle sous ~60 s."
    ;;
  *) echo "Commande inconnue: $cmd (demo|up|down|status)"; exit 1 ;;
esac
