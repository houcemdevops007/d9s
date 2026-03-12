# d9s - Docker TUI (Terminal User Interface)

**Auteur** : KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD

**d9s** est une interface utilisateur en terminal (TUI) puissante, rapide et intuitive pour gérer de bout en bout l'écosystème Docker. Conçu spécifiquement pour les ingénieurs DevOps, SecOps et les développeurs, `d9s` combine la gestion de conteneurs, le pilotage de Docker Compose, et l'intégration de scanners de sécurité avancés (Trivy et Snyk) au sein d'une seule interface navigable entièrement au clavier.

## 🚀 Fonctionnalités Principales

### 1. Gestion Multi-Hôtes Distants (TCP/Unix)
`d9s` ne se limite pas au démon Docker local (socket `unix:///var/run/docker.sock`). Il permet de gérer dynamiquement plusieurs hôtes Docker distincts via un système de **contextes**.
- **Configuration dynamique** : En renseignant le fichier `~/.config/d9s/config.json`, vous pouvez enregistrer des hôtes distants (exemple : `tcp://172.16.50.13:2375`).
- **Basculement instantané** : Grâce au panneau latéral "CONTEXTS", l'utilisateur navigue et bascule d'un hôte Docker vers un autre (local, distant, serveurs de production, etc.) d'un simple appui sur la touche **Entrée** sans jamais redémarrer l'application.

### 2. Contrôle et Visualisation de l'Écosystème Docker
`d9s` permet de tout voir et tout faire de manière centralisée, avec des raccourcis intuitifs :
- **Conteneurs (touche `c`)** : Lister les conteneurs (actifs/inactifs), voir leur statut. Raccourcis pour démarrer (`s` ? non, `x` pour stop, `r` pour restart, `R` ou `Suppr` pour supprimer).
- **Shell Interactif (touche `S`)** : Ouvrir un shell interactif directement à l'intérieur d'un conteneur en cours d'exécution.
- **Images (touche `g` ou `i`)** : Voir les images locales.
- **Volumes (touche `v`)** & **Réseaux (touche `n`)** : Gérer les volumes et les réseaux avec vue d'ensemble.
- **Inspect (touche `i`)** : Obtenir un dump JSON clair et formatté ou les détails d'un objet ciblé.
- **Logs (touche `l`)** : Consulter les journaux en direct des conteneurs.
- **Events & Stats** : Surveiller en temps réel ce qui se passe sur les ressources allouées (CPU/RAM).

### 3. Gestion Native de Docker Compose (Projets)
Le panneau "PROJECTS" gère intelligemment la couche Docker Compose :
- Détecte automatiquement les environnements (dossiers Compose de la machine ou variables d'environnement).
- Pilote `docker-compose` avec des raccourcis efficaces :
  - `u` = Compose Up
  - `d` = Compose Down
  - `p` = Compose Pull
  - `b` = Compose Build

### 4. SecOps & Conformité (Trivy + Snyk)
La plus grande force de `d9s` réside dans ses onglets de sécurité intégrés nativement. Sélectionnez une Image Docker (ou un conteneur) et :
- **Trivy Scan** : Lance un rapport de vulnérabilités open-source rapide et détaille les failles (Critique, Haut, Moyen, Bas) par package avec le numéro de CVE.
- **Snyk Scan** : Utilise l'outil professionnel Snyk (si installé localement) pour une analyse profonde de vos images, avec un affichage des vulnérabilités classées.
- **Best Practices** : Moteur interne de vérification de bonnes pratiques (heuristiques sur l'utilisateur root, la version LTS, l'étiquetage réseau, les ports ouverts, la taille excessive...) croisant les résultats d'inspection et de sécurité de l'image.

## 💻 Raccourcis Clavier (Keyboard Shortcuts)
Toute la TUI est gérée avec le clavier sans aucune dépendance à la souris pour optimiser la productivité globale. 
Les touches comme `Tab` permettent de basculer de panneaux/onglets, les `Flèches` pour naviguer dans une liste. Le `/` effectue des recherches instantanées filtrées dans les listes actives.

## 🛠️ Compilation Multi-Plateforme (Linux & MacOS)

Le projet intègre un Makefile conçu pour simplifier la compilation vers de multiples architectures.

**Pour compiler depuis les sources (Go `1.22+` requis) :**
```bash
# Compiler pour votre système local (Mac ou Linux)
make build

# Compiler spécialement pour Linux (AMD64 et ARM64)
make build-linux

# Compiler spécialement pour MacOS (AMD64 et ARM64)
make build-darwin

# Tout compiler d'un seul coup
make build-all
```
Tous les binaires seront exportés dans le dossier `/build`.

**Pour installer :**
```bash
sudo make install
# d9s se copiera dans /usr/local/bin/d9s
```

> Note : Les CLI externes comme `trivy` et `snyk` doivent être installées et ajoutées à la variable de chemin `$PATH` du système d'exploitation tournant `d9s` (MacOS ou Linux) afin que les onglets de Scan fonctionnent correctement.
