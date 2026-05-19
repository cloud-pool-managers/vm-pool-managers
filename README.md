# VM Pool Managers

Plateforme de gestion de pools de machines virtuelles pour l'enseignement. Permet aux professeurs de creer, planifier et superviser des pools de VMs OpenStack attribuees automatiquement aux etudiants.

## Architecture

```
┌─────────────┐       gRPC-Web        ┌─────────────────┐        gRPC         ┌──────────────────────┐
│   Frontend  │ ─────────────────────> │  Control Center │ ──────────────────> │ Microservice OpenStack│
│  (SvelteKit)│ <───── streams ─────── │    (Go/gRPC)    │ <────────────────── │   (Go/gophercloud)    │
└─────────────┘                        └────────┬────────┘                     └──────────────────────┘
                                                │
                                    PostgreSQL + LISTEN/NOTIFY
                                                │
                                        ┌───────┴───────┐
                                        │  PostgreSQL   │
                                        └───────────────┘
```

### Composants

| Composant | Chemin | Role |
|-----------|--------|------|
| **Frontend** | `frontend/` | Interface web SvelteKit 5 + Tailwind CSS. Communique via gRPC-Web. |
| **Control Center** | `control_center/` | Serveur gRPC central. Gere les pools, users, configs, attribution des VMs, monitoring SSH. |
| **Microservice OpenStack** | `microservices/openstack/` | Worker qui provisionne/detruit les VMs via l'API OpenStack (gophercloud). |
| **Caddy** | `caddy/` | Reverse proxy qui translate gRPC-Web, expose le frontend et l'API REST sur le port 80. |
| **Cloud-init** | `cloud-init-postgres.yaml` | Script d'initialisation pour la VM hebergeant PostgreSQL. |

### Communication

- **Frontend <-> Control Center** : gRPC-Web via Caddy (port 80 `/rpc/*`)
- **Control Center <-> Microservice** : gRPC direct (port 50052)
- **Control Center -> Frontend** : Streaming gRPC (updates temps reel via PostgreSQL LISTEN/NOTIFY)
- **VMs -> Control Center** : API REST (port 50055 `/api/*`) pour le registrar et le monitoring SSH

## Prerequis

- Go 1.22+
- Node.js 20+ / npm
- PostgreSQL 15+
- Docker (pour Caddy)
- [Task](https://taskfile.dev) (task runner)
- Acces a un cloud OpenStack avec 2 projets (infra + VMs)

## Installation

### 1. Cloner et configurer

```sh
git clone <repo-url>
cd vm-pool-managers
task setup
```

Le script interactif (`config.go`) cree les fichiers `.env` avec les parametres necessaires.

### 2. OpenStack — clouds.yaml

Copier le template et remplir les credentials :

```sh
mkdir -p ~/.config/openstack
cp clouds.yaml.template ~/.config/openstack/clouds.yaml
# Editer et remplir les application credentials
```

Deux projets sont necessaires :
- **ipp-idcs-vmpoolmanager** : listing des images, flavors, networks (projet infra)
- **ipp-idcs-vmpool** : creation/suppression des VMs etudiants

### 3. PostgreSQL

Option A — VM dediee avec cloud-init :
```sh
# Lancer une instance Ubuntu 24.04 sur OpenStack avec cloud-init-postgres.yaml
```

Option B — Installation manuelle :
```sh
sudo -i -u postgres psql
```
```sql
CREATE USER admin WITH PASSWORD 'votre_mot_de_passe';
CREATE DATABASE control_center OWNER admin;
\c control_center
ALTER SCHEMA public OWNER TO admin;
GRANT ALL ON SCHEMA public TO admin;
```

Appliquer le schema du registrar :
```sh
psql -h localhost -U admin -d control_center < sql/registrar_schema.sql
```

### 4. Caddy (reverse proxy)

```sh
cd caddy
docker build -t my-caddy .
docker run -d -p 80:80 --add-host=host.docker.internal:host-gateway --name caddy my-caddy
```

### 5. Lancer le projet

```sh
task dev
```

Lance via tmux :
- **backend** : microservice OpenStack (port 50052)
- **control** : control center (port 50051 gRPC + 50055 REST)
- **frontend** : SvelteKit dev server (port 5173)

## Configuration (.env)

Voir `.env.example` pour toutes les variables. Les fichiers `.env` sont a placer :
- `.env` — racine (utilise par `task setup` et le microservice)
- `control_center/.env` — specifique au control center
- `microservices/openstack/.env` — specifique au microservice

Variables importantes :

| Variable | Description |
|----------|-------------|
| `POSTGRES_*` | Connexion PostgreSQL |
| `OS_CLOUD` | Nom du cloud OpenStack pour les VMs |
| `INFRA_OS_CLOUD` | Nom du cloud pour le listing infra |
| `SSH_PRIVATE_KEY_PATH` | Cle SSH pour le monitoring d'activite |
| `SKIP_RCLONE` | `true` pour desactiver rclone en dev |
| `REGISTRAR_CONTROL_CENTER_URL` | URL du control center pour les VMs |

## Fonctionnalites

### Gestion des pools
- Creation de pools avec image, flavor, network, min/max VMs
- Planification horaire (jour/heure de debut + duree)
- Jours off configurables
- Suppression automatique en fin de fenetre horaire
- Replanification hebdomadaire

### Attribution des VMs
- Attribution automatique aux etudiants via cle SSH
- Un etudiant = un serveur dans le pool
- Injection de cle SSH via cloud-init + setup post-boot
- Username genere : `student_<pool_id>`

### Monitoring
- Detection d'activite SSH (connexions actives)
- Heartbeat des VMs via registrar (auto-enregistrement)
- Reaper : marque les VMs sans heartbeat comme "dead"
- Inventaire temps reel avec status par VM

### Partage de fichiers (rclone)
- Mount SFTP bidirectionnel etudiant <-> professeur
- `~/depot` sur VM etudiant -> accessible par le professeur
- `~/shared_files` sur VM etudiant <- fichiers du professeur

### Configs
- Configurations reutilisables (scripts cloud-init personnalises)
- Associees aux pools pour le provisionnement

## Commandes Task

| Commande | Description |
|----------|-------------|
| `task dev` | Lance tout via tmux |
| `task backend` | Lance le microservice seul |
| `task control` | Lance le control center seul |
| `task frontend` | Lance le frontend (port 80, sudo) |
| `task frontexpose` | Lance le frontend sur le reseau (port 5173) |
| `task setup` | Configuration interactive initiale |
| `task build` | Build le microservice |
| `task clean` | Supprime les builds et DB SQLite |
| `task cli` | Ouvre le CLI OpenStack |

## Structure du projet

```
vm-pool-managers/
├── frontend/               # SvelteKit 5 + Tailwind + gRPC-Web
│   ├── src/
│   │   ├── lib/
│   │   │   ├── grpc/       # Client gRPC et types protobuf
│   │   │   ├── components/ # Composants Svelte
│   │   │   ├── store/      # Stores Svelte (auth, pools)
│   │   │   └── utils/      # Handlers de mise a jour
│   │   └── routes/         # Pages SvelteKit
│   └── proto/              # Fichier .proto frontend
├── control_center/         # Serveur gRPC central (Go)
│   ├── grpc/               # Serveurs gRPC + inventory REST
│   ├── internal/
│   │   ├── attribvm/       # Attribution des VMs aux etudiants
│   │   ├── auth/           # Authentification
│   │   ├── monitoring/     # Monitoring pools + SSH
│   │   ├── pool/           # CRUD pools
│   │   ├── rclone/         # Setup partage fichiers
│   │   └── sshinject/      # Injection cles SSH
│   └── models/             # Modeles GORM
├── microservices/openstack/ # Worker OpenStack (Go)
│   ├── internal/
│   │   ├── jobs/           # Jobs de creation/suppression VMs
│   │   └── worker/         # Pool de workers
│   ├── models/             # Client OpenStack
│   └── utils/              # Helpers OpenStack
├── caddy/                  # Reverse proxy (Dockerfile + Caddyfile)
├── sql/                    # Schemas SQL
├── Taskfile.yaml           # Task runner
├── .env.example            # Template de configuration
├── clouds.yaml.template    # Template OpenStack
└── cloud-init-postgres.yaml # Cloud-init pour la DB
```

## Reinitialiser

### Base PostgreSQL
```sh
psql -h localhost -U admin -d control_center -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
psql -h localhost -U admin -d control_center < sql/registrar_schema.sql
```

### SQLite microservice
```sh
task clean
```

## Pistes d'amelioration

- Authentification OIDC + OpenPubkey
- Interface drag & drop pour le partage de fichiers
- Broker CI integre
- Messages d'erreur detailles dans les reponses protobuf
- Dashboard de monitoring avance (Grafana)
