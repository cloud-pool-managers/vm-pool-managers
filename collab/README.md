# Spike — VS Code collaboratif temps réel (Open Collaboration Tools)

Prototype d'**édition simultanée multi-utilisateurs** dans VS Code en ligne (code-server),
avec curseurs/sélections en direct et synchronisation **CRDT**.

## Pourquoi Open Collaboration Tools (OCT)

- **Microsoft Live Share ne fonctionne pas dans code-server** (propriétaire, lié à un compte
  MS, indisponible sur VS Code OSS / Open VSX).
- **OCT** (projet Eclipse, ex-TypeFox) est la seule solution **open-source** de live-share
  qui marche dans **code-server / VS Code OSS / Theia**. Extension publiée sur Open VSX
  (`typefox.open-collaboration-tools`) + **serveur auto-hébergeable** (aucune donnée chez un tiers).
- code-server est **déjà déployé** sur les VMs étudiantes (port 8443) → intégration naturelle.

## Architecture du prototype

```
 code-server-host  (VS Code A, :8443) ─┐
                                        ├─ websocket ─▶  oct-server (:8100)
 code-server-guest (VS Code B, :8444) ─┘                (CRDT, sessions en mémoire)
```

- 2 instances code-server **séparées** = 2 utilisateurs distincts.
- L'extension OCT (dans chaque code-server) pointe sur notre serveur OCT auto-hébergé
  (`settings.json` → `oct.serverUrl`), pas sur l'instance publique `open-collab.tools`.
- Auth serveur : **login simple par pseudo** (`OCT_ACTIVATE_SIMPLE_LOGIN`), sans OAuth, pour le proto.

## Tester

```bash
cd collab
docker compose up -d        # 1er run : npx télécharge le serveur OCT + install de l'extension
```

1. **Hôte** → http://localhost:8443 (VS Code A, ouvre `demo.py`).
   Palette de commandes (`F1`) → **« Open Collaboration: Share »** → entre un pseudo →
   un **code de session** (room token) est copié/affiché.
2. **Invité** → http://localhost:8444 (VS Code B).
   Palette → **« Open Collaboration: Join Collaboration Session »** → colle le code.
3. → Vous éditez **le même fichier en temps réel**, avec les **curseurs/sélections** de l'autre
   visibles. (Le guest voit le workspace partagé par l'hôte.)

Arrêt : `docker compose down`.

> Le serveur auto-hébergé est déjà câblé via `settings.json` (`oct.serverUrl=http://oct-server:8100/`
> + `oct.alwaysAskToOverrideServerUrl=false`) → aucune URL à saisir. Auth = un simple **pseudo**.
> Pour rejoindre, utiliser le **code de session** (commande « Join »), pas le lien web public.

## Intégration plateforme (prochaine étape, hors proto)

Modèle « session collaborative » dans CloudPoolManager :

1. **Serveur OCT** : un service `oct-server` ajouté à la stack (sur la VM de contrôle ou dédié),
   derrière Caddy en TLS, auth via le **même OIDC/Dex** (OCT supporte OAuth générique/Keycloak)
   → un seul login pour la plateforme et la collaboration.
2. **VMs étudiantes** : le cloud-init (`scripts/nbgrader-cloud-init.sh`) installe l'extension
   `typefox.open-collaboration-tools` dans code-server et fixe `oct.serverUrl` vers le serveur OCT.
3. **Control center** : notion de *session collaborative* (N membres → partage d'un workspace) :
   - soit N étudiants rejoignent la session de l'enseignant (revue de code, TP guidé) ;
   - soit un pool « binôme/groupe » où les membres co-éditent.
   Un bouton « Démarrer/Rejoindre une session collaborative » dans l'UI, qui relaie le code de session.

## Décision / go-no-go

- **Go** sur OCT : c'est la voie réaliste pour du vrai temps réel dans code-server, auto-hébergeable
  (pas de dépendance externe, données maîtrisées), et qui réutilise l'existant (code-server + OIDC).
- Limite connue : le serveur OCT garde les sessions **en mémoire** (pas de scaling horizontal) —
  suffisant pour l'échelle d'un cours ; à surveiller si beaucoup de sessions simultanées.
