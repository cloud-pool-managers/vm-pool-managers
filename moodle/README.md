# Moodle local (développement)

Moodle complet (interface web + Web Services) pour développer l'intégration
CloudPoolManager↔Moodle sans dépendre du Moodle de l'école.

> Images : `bitnamilegacy/moodle:5.0.2` + `bitnamilegacy/mariadb:11.8.3`.
> (Bitnami a déplacé ses tags versionnés vers le namespace `bitnamilegacy` en 2025.)

## Démarrage

```bash
cd moodle
cp .env.example .env        # renseigner les mots de passe
docker compose up -d        # ~quelques minutes au 1er boot (installation Moodle)
cd .. && scripts/moodle-bootstrap.sh
```

- **Interface Moodle** : http://localhost:8081 — login `admin` / `MOODLE_ADMIN_PASSWORD`.
  Tu y as TOUT Moodle : cours, inscriptions, carnet de notes, devoirs, plugins…
- `scripts/moodle-bootstrap.sh` (idempotent) :
  - active les **Web Services** + le protocole **REST** ;
  - crée un service externe dédié `cpm_service` + un **token** pour l'admin ;
  - crée 2 cours de démo (`CPM-PY101`, `CPM-DS200`), 4 élèves + 1 prof, inscrits ;
  - écrit `MOODLE_URL` et `MOODLE_TOKEN` dans le `.env` racine (consommés par le control center).

Comptes de démo : élèves `alice/bob/charlie/diana`, prof `prof1` — mot de passe `Student_2026!`.

## Tester un appel Web Service

```bash
source <(grep -E '^MOODLE_(URL|TOKEN)=' ../.env)
curl -s "$MOODLE_URL/webservice/rest/server.php" \
  --data-urlencode "wstoken=$MOODLE_TOKEN" \
  --data-urlencode "wsfunction=core_enrol_get_enrolled_users" \
  --data-urlencode "courseid=2" --data-urlencode "moodlewsrestformat=json" | python3 -m json.tool
```

## Sécurité ⚠️

- `moodle/.env` (mots de passe) et le `MOODLE_TOKEN` du `.env` racine sont **gitignorés**
  (`*.env`) — jamais commités. Le dépôt ne contient que `.env.example` (placeholders).
- `bootstrap.php` crée des comptes de démo avec un mot de passe par défaut : **dev uniquement**.

## Réinitialiser

```bash
cd moodle && docker compose down -v   # -v supprime les volumes (DB + données Moodle)
```
