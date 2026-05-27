# CloudPoolManager

Plateforme de gestion de pools de machines virtuelles OpenStack pour l'enseignement.  
Permet aux enseignants de créer, planifier et superviser des pools de VMs attribuées automatiquement aux étudiants, avec accès terminal web via Apache Guacamole.

---

## Architecture

```
Navigateur (étudiant / admin)
        │
        │ HTTPS (443)
        ▼
┌──────────────┐
│    Caddy     │  Reverse proxy — route gRPC-Web, REST, Guacamole, Frontend
└──────┬───────┘
       │
       ├─── /rpc/*     gRPC-Web ──► Control Center :50051
       ├─── /api/*     REST     ──► Control Center :50055
       ├─── /guacamole/*        ──► Guacamole      :18080 (tunnel SSH)
       └─── /*         SPA      ──► SvelteKit      :5173
                                         │
                              ┌──────────┴──────────┐
                              │   Control Center     │
                              │       (Go)           │
                              └──┬──────────────┬───┘
                                 │              │
                          gRPC :50052    PostgreSQL
                                 │        LISTEN/NOTIFY
                                 ▼
                     ┌──────────────────────┐
                     │ Microservice OpenStack│
                     │   (Go/gophercloud)   │
                     └──────────┬───────────┘
                                │
                          OpenStack API
                         (création VMs)
```

### Services

| Service | Chemin | Port | Rôle |
|---------|--------|------|------|
| **Frontend** | `frontend/` | 5173 / 80 | SvelteKit 5 + Tailwind. UI admin et portail étudiant |
| **Control Center** | `control_center/` | 50051 (gRPC) 50055 (REST) | Orchestration centrale : pools, users, attribution, monitoring |
| **Microservice OpenStack** | `microservices/openstack/` | 50052 | Provisionnement/suppression VMs via gophercloud |
| **Caddy** | `caddy/` | 443 | Reverse proxy TLS — route gRPC-Web, REST et Guacamole |
| **GLAuth** | `auth/glauth.cfg` | 3893 | Annuaire LDAP léger — stocke users et groupes |
| **Dex** | `auth/dex.yaml` | 5556 | Fournisseur OIDC — émet des JWT signés depuis GLAuth |
| **Guacamole** | VM `infra-postgres` | 18080 (tunnel) | Gateway SSH-in-browser — terminal web pour les étudiants |

---

## Flux d'authentification

```
Navigateur          Frontend          Dex (OIDC)         GLAuth (LDAP)
    │                   │                  │                    │
    │  clic "Se connecter"│                 │                    │
    │──────────────────►│                  │                    │
    │                   │  redirect PKCE   │                    │
    │◄──────────────────│─────────────────►│                    │
    │  login page Dex   │                  │                    │
    │──────────────────────────────────────┤                    │
    │  user:pass        │                  │  LDAP bind+search  │
    │                   │                  │───────────────────►│
    │                   │                  │◄───────────────────│
    │                   │                  │  émission JWT      │
    │◄──────────────────────────────────── │                    │
    │  code PKCE        │                  │                    │
    │──────────────────►│                  │                    │
    │                   │  échange code    │                    │
    │                   │─────────────────►│                    │
    │                   │◄─────────────────│                    │
    │                   │  id_token (JWT)  │                    │
    │                   │  stocké en mémoire                    │
```

Le JWT est envoyé dans chaque requête gRPC comme `Authorization: Bearer <token>`.  
Le Control Center valide la signature et extrait le rôle (`admin` ou `user`).

**Comptes GLAuth** (fichier `auth/glauth.cfg`) :
- Groupe `admins` (gidnumber 5501) → rôle `admin` dans l'app
- Groupe `users` (gidnumber 5502) → rôle `user`

---

## Flux VM → Terminal web (Guacamole)

```
1. Pool créé          Control Center provisionne des VMs OpenStack
                      (cloud-init installe vmuser + clé SSH du projet)

2. VM ACTIVE          Boucle monitoring toutes les 30s :
                      → Appel API REST Guacamole : CreateSSHConnection(name, ip)
                      → Guacamole stocke IP + clé privée SSH
                      → guac_connection_id sauvegardé en DB (vm_instances)

3. Étudiant entre     Frontend appelle /api/guac-url?ip=<vm_ip>
   sa clé SSH         Control Center construit l'URL :
                      /guacamole/#/client/<base64(connID+\x00c\x00mysql)>?token=<admin_token>

4. Clic "Terminal"    Navigateur → Caddy → Guacamole (tunnel SSH 18080)
                      Guacamole ouvre SSH vers la VM avec sa clé privée stockée
                      → Terminal dans le navigateur, sans client SSH
```

La clé privée SSH ne quitte jamais le serveur. L'étudiant n'a pas à se connecter à Guacamole.

---

## Prérequis

- Go 1.22+
- Node.js 20+ / npm
- PostgreSQL 15+
- Docker + Docker Compose
- [Task](https://taskfile.dev) (`brew install go-task`)
- Accès à un cloud OpenStack avec 2 projets

---

## Installation

### 1. Cloner et configurer

```sh
git clone <repo-url>
cd vm-pool-managers
cp .env.example .env
# Éditer .env avec vos valeurs
```

### 2. OpenStack — clouds.yaml

```sh
mkdir -p ~/.config/openstack
cp clouds.yaml.template ~/.config/openstack/clouds.yaml
# Remplir les application credentials
```

Deux projets nécessaires :
- **ipp-idcs-vmpoolmanager** — listing images/flavors/networks (infra)
- **ipp-idcs-vmpool** — création/suppression des VMs étudiants

### 3. PostgreSQL

```sql
CREATE USER admin WITH PASSWORD 'votre_mot_de_passe';
CREATE DATABASE control_center OWNER admin;
\c control_center
ALTER SCHEMA public OWNER TO admin;
GRANT ALL ON SCHEMA public TO admin;
```

Appliquer le schéma du registrar :

```sh
psql -h localhost -U admin -d control_center < sql/registrar_schema.sql
```

### 4. Authentification (GLAuth + Dex)

Les fichiers de config sont dans `auth/` (gitignorés — ne pas committer).

```sh
# Copier les templates
cp auth/glauth.cfg.example auth/glauth.cfg
cp auth/dex.yaml.example auth/dex.yaml
# Éditer les mots de passe (SHA256 pour GLAuth)

task auth   # Lance GLAuth :3893 + Dex :5556 via Docker Compose
```

Générer un hash SHA256 pour un mot de passe :
```sh
echo -n "monmotdepasse" | sha256sum
```

### 5. Guacamole

Guacamole tourne sur la VM `infra-postgres` (157.136.249.205).  
Comme le port 8080 est protégé par le security group OpenStack, on passe par un tunnel SSH :

```sh
task guac:tunnel   # Lance le tunnel en avant-plan (127.0.0.1:18080 → VM:8080)
# Dans un autre terminal :
task control       # Le control center utilise GUACAMOLE_URL=http://127.0.0.1:18080/guacamole
```

Pour redéployer Guacamole sur la VM :
```sh
ssh ubuntu@157.136.249.205
docker compose -f ~/docker-compose.guacamole.yaml --env-file ~/.env.guac up -d
```

### 6. Caddy (reverse proxy)

```sh
task caddy
```

Reconstruit l'image Docker et redémarre Caddy sur le port 443.

### 7. Lancer le projet complet

```sh
task guac:tunnel &  # tunnel Guacamole
task auth           # GLAuth + Dex
task dev            # backend + control + frontend via tmux
```

---

## Configuration (.env)

| Variable | Description |
|----------|-------------|
| `POSTGRES_*` | Connexion PostgreSQL centrale |
| `OS_CLOUD` | Projet OpenStack pour les VMs étudiants |
| `INFRA_OS_CLOUD` | Projet OpenStack pour le listing infra |
| `SSH_PUBLIC_KEY_PATH` | Clé publique injectée dans les VMs |
| `SSH_PRIVATE_KEY_PATH` | Clé privée pour monitoring SSH + Guacamole |
| `GUACAMOLE_URL` | URL interne Guacamole (via tunnel : `http://127.0.0.1:18080/guacamole`) |
| `GUACAMOLE_ADMIN_USER` | Login admin Guacamole |
| `GUACAMOLE_ADMIN_PASS` | Mot de passe admin Guacamole |
| `GUACAMOLE_SSH_USER` | Utilisateur Linux sur les VMs (`vmuser`) |
| `REGISTRAR_CONTROL_CENTER_URL` | URL REST du control center pour les VMs |
| `SKIP_RCLONE` | `true` pour désactiver rclone en dev local |

---

## Commandes Task

| Commande | Description |
|----------|-------------|
| `task dev` | Lance tout (backend + control + frontend) via tmux |
| `task backend` | Lance le microservice OpenStack seul |
| `task control` | Lance le control center seul |
| `task frontend` | Lance le frontend (port 80, sudo) |
| `task frontexpose` | Lance le frontend sur le réseau (port 5173) |
| `task auth` | Lance GLAuth + Dex via Docker Compose |
| `task auth:stop` | Arrête GLAuth + Dex |
| `task guac:tunnel` | Tunnel SSH Guacamole (127.0.0.1:18080 → infra-postgres:8080) |
| `task guac` | Lance Guacamole localement (optionnel, pour le dev) |
| `task caddy` | Rebuild et redémarre Caddy |
| `task setup` | Configuration interactive initiale |
| `task build` | Build le microservice OpenStack |
| `task clean` | Supprime les builds et la DB SQLite |

---

## Structure du projet

```
vm-pool-managers/
├── frontend/                    # SvelteKit 5 + Tailwind + gRPC-Web
│   ├── src/
│   │   ├── lib/
│   │   │   ├── grpc/            # Clients gRPC générés (protobuf)
│   │   │   │   └── transport.ts # Transport authentifié (Bearer JWT)
│   │   │   ├── components/      # Modales et composants UI
│   │   │   ├── store/           # authStore, serverpoolStore
│   │   │   └── index.ts         # loadAll(), logout(), subscribeUserUpdate()
│   │   └── routes/
│   │       ├── +layout.svelte   # Nav + garde d'auth globale
│   │       ├── +page.svelte     # Portail étudiant (clé SSH → VM + terminal)
│   │       ├── login/           # Page de connexion SSO
│   │       ├── auth/callback/   # Callback OIDC PKCE
│   │       ├── serverpool/      # Gestion des pools (admin)
│   │       ├── inventory/       # Inventaire temps réel des VMs (admin)
│   │       ├── config/          # Configurations cloud-init (admin)
│   │       └── profile/         # Profil utilisateur + clé SSH perso
│   └── proto/                   # frontcontrol.proto (frontend)
│
├── control_center/              # Serveur gRPC central (Go)
│   ├── grpc/
│   │   ├── server.go            # Bootstrap gRPC + HTTP mux (50051/50055)
│   │   ├── inventory.go         # /api/inventory + /api/guac-url
│   │   └── client_openstack.go  # Sync servers OpenStack → PostgreSQL
│   ├── internal/
│   │   ├── attribvm/            # Attribution VM → étudiant par clé SSH
│   │   ├── auth/                # Service AuthenticateUser (OIDC JWT)
│   │   ├── configpool/          # CRUD configurations cloud-init
│   │   ├── gatherdata/          # Proxy listing images/flavors/networks
│   │   ├── guacamole/           # Client API REST Guacamole (token caché)
│   │   ├── monitoring/          # Heartbeat, SSH activity, sync Guacamole
│   │   ├── oidc/                # Middleware JWT — validation + extraction rôle
│   │   ├── pool/                # CRUD pools + scheduling
│   │   ├── rclone/              # Setup partage fichiers SFTP
│   │   ├── sshinject/           # Injection clé SSH post-boot via SSH
│   │   └── user/                # CRUD utilisateurs + streaming updates
│   ├── models/                  # Modèles GORM (Server, Serverpool, VMInstance…)
│   └── frontcontrolpb/          # Code généré protobuf (Go)
│
├── microservices/openstack/     # Worker OpenStack (Go)
│   ├── internal/
│   │   ├── jobs/                # Logique création/suppression VM
│   │   └── worker/              # Pool de goroutines
│   └── models/                  # Client gophercloud
│
├── caddy/                       # Reverse proxy
│   ├── Caddyfile                # Routes : gRPC-Web, REST, Guacamole, SPA
│   └── Dockerfile
│
├── sql/
│   └── registrar_schema.sql     # Table vm_instances + colonne guac_connection_id
│
├── proto/
│   ├── frontcontrol.proto       # Frontend ↔ Control Center
│   └── poolmanager.proto        # Control Center ↔ Microservice
│
├── Taskfile.yaml                # Toutes les commandes de dev
├── .env.example                 # Template de configuration
├── clouds.yaml.template         # Template OpenStack credentials
├── docker-compose.auth.yaml     # GLAuth + Dex
└── cloud-init-postgres.yaml     # Cloud-init pour VM PostgreSQL
```

---

## Fonctionnalités

### Gestion des pools (admin)
- Création avec image, flavor, réseau, min/max VMs, script cloud-init
- Planification horaire : jour/heure de début + durée de la fenêtre
- Suppression automatique en fin de fenêtre, replanification hebdomadaire
- Ajout d'étudiants par CSV (login + clé SSH publique)

### Attribution automatique (étudiant)
- L'étudiant colle sa clé SSH publique sur le portail
- Le système retrouve les pools où cette clé est enregistrée
- Une VM disponible lui est assignée, sa clé SSH injectée via SSH
- IP + commande SSH + lien terminal web retournés immédiatement

### Terminal web (Guacamole)
- Connexion SSH via navigateur, sans client SSH
- Session ouverte automatiquement, sans login Guacamole
- Bouton "Terminal web" affiché directement sur le portail étudiant

### Monitoring (admin)
- Détection d'activité SSH (connexions actives sur les VMs)
- Heartbeat via vm-registrar (auto-enregistrement des VMs au démarrage)
- Inventaire temps réel : statut, santé, dernière activité, lien terminal

### Partage de fichiers (rclone)
- Mount SFTP bidirectionnel étudiant ↔ enseignant
- `~/depot` sur la VM étudiant → visible par l'enseignant
- `~/shared_files` sur la VM étudiant ← fichiers de l'enseignant

---

## Réinitialisation

```sh
# Remettre à zéro la DB PostgreSQL
psql -h localhost -U admin -d control_center \
  -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO admin;"
psql -h localhost -U admin -d control_center < sql/registrar_schema.sql

# Supprimer la DB SQLite du microservice
task clean
```

---

## Notes d'infrastructure

- **Disque VM infra-postgres** : le control center déployé sur cette VM génère des logs verbeux dans `/tmp/control.log`. Un cron root tronque ce fichier toutes les 5 minutes pour éviter le remplissage du disque (19 Go).
- **Tunnel Guacamole** : le port 8080 de `infra-postgres` est fermé dans le security group OpenStack. `task guac:tunnel` ouvre `127.0.0.1:18080 → 157.136.249.205:8080` via SSH.
- **Token Guacamole** : le client Go met le token d'auth Guacamole en cache 50 minutes pour éviter une requête d'authentification par VM lors du chargement de l'inventaire.
