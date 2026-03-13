# Notion Manager - OpenClaw Skill

Un skill OpenClaw pour créer et gérer des pages, bases de données et blocs Notion via *notion-cli* et l’API Notion.

##  Installation et Configuration

### Prérequis

- **notion-cli** : `npm install -g @iansinnott/notion-cli`
- Une intégration Notion : créez-la sur https://notion.so/my-integrations
- Clé API (commence par *ntn_* ou *secret_*)

### Variable d’Environnement Requise

```bash
export NOTION_TOKEN=$(cat ~/.config/notion/api_key)
```

Ou configurez le fichier :

```bash
mkdir -p ~/.config/notion
echo "ntn_your_key_here" > ~/.config/notion/api_key
```

**Important :** Partager les pages et bases de données cibles avec votre intégration (menu "..." → "Connecter à" → nom de l’intégration).

##  Guide de Démarrage Rapide

### Exemple 1 : Rechercher des pages

```
User: "Recherche les pages avec le titre 'AIStories'"
→ notion-cli search --query "AIStories"
```

### Exemple 2 : Récupérer une page

```
User: "Affiche la page avec l’ID 2fdf172c-0949-80dd-b83b-c1df0410d91b"
→ notion-cli page retrieve 2fdf172c-0949-80dd-b83b-c1df0410d91b
```

### Exemple 3 : Interroger une base de données

```
User: "Liste les éléments de la base dont le statut est Backlog"
→ notion-cli db query <DB_ID> -a '{"property":"Status","status":{"equals":"Backlog"}}'
```

### Exemple 4 : Créer une page dans une base

```
User: "Crée une page 'Nouvelle idée' dans la base X"
→ POST https://api.notion.com/v1/pages
   parent: { database_id: "..." }
   properties.Name.title[0].text.content: "Nouvelle idée"
```

### Exemple 5 : Mettre à jour le statut d’une page

```
User: "Passe la page Y en 'In progress'"
→ PATCH https://api.notion.com/v1/pages/PAGE_ID
   properties.Status.status.name: "In progress"
```

##  Exemples de Cas d’Usage

### Cas 1 : Recherche et lecture

```
# Rechercher
User: "Trouve les pages contenant 'roadmap'"

# Détail d’une page
User: "Affiche le contenu (blocs) de la page 2fdf172c-0949-80dd-b83b-c1df0410d91b"
→ notion-cli page retrieve <PAGE_ID> -r

# Infos d’une base
User: "Donne les infos de la base de données <DB_ID>"
→ notion-cli db retrieve <DB_ID>
```

### Cas 2 : Bases de données et filtres

```
# Filtrer par statut
User: "Liste les éléments de la base dont le statut est 'Active'"
→ notion-cli db query <DB_ID> -a '{"property":"Status","status":{"equals":"Active"}}'

# Mode interactif (requêtes complexes)
User: "Lance une requête interactive sur la base <DB_ID>"
→ notion-cli db query <DB_ID>
```

### Cas 3 : Création et mise à jour de pages

```
# Créer une page avec titre
User: "Crée une page 'Nouvelle idée' dans la base X"

# Mettre à jour titre, statut, priorité, date
User: "Modifie la page Y : titre 'Nouveau titre', statut 'In progress', priorité High, date 2026-02-10"
→ PATCH /v1/pages/PAGE_ID avec properties (Name, Status, Priority, Due date, Description)
```

##  Fonctionnalités Principales

### Pages
-  Recherche de pages et bases
-  Récupération d’une page (avec ou sans blocs)
-  Création de pages dans une base
-  Mise à jour des propriétés (titre, statut, date, etc.)

### Bases de données
-  Récupération des métadonnées (db retrieve)
-  Requêtes avec filtres (db query, option `-a`)
-  Mode interactif pour requêtes complexes
-  Formats de sortie : table, csv, json, yaml (--raw pour JSON brut)

### Types de propriétés courants
- **Title** : `{"title": [{"text": {"content": "..."}}]}`
- **Rich text** : `{"rich_text": [{"text": {"content": "..."}}]}`
- **Status** : `{"status": {"name": "Option"}}`
- **Select** : `{"select": {"name": "Option"}}`
- **Multi-select** : `{"multi_select": [{"name": "A"}, {"name": "B"}]}`
- **Date** : `{"date": {"start": "YYYY-MM-DD", "end": "..."}}`
- **Checkbox** : `{"checkbox": true}`
- **Number** : `{"number": 42}`
- **URL** : `{"url": "https://..."}`

##  Notes Techniques

- Les ID de pages et bases sont des UUID (avec ou sans tirets).
- L’authentification repose sur la variable **NOTION_TOKEN**.
- Utiliser l’en-tête `Notion-Version: 2025-09-03` (ou la version courante) pour les appels API directs.
- Les limites de débit sont gérées par le CLI / l’API.

##  Ressources

- **notion-cli** : https://github.com/litencatt/notion-cli
- **Documentation API Notion** : https://developers.notion.com
- **Créer une intégration** : https://notion.so/my-integrations

##  Dépannage

### Erreur d’authentification
**Cause :** `NOTION_TOKEN` absent ou invalide.  
**Solution :** Vérifiez que la variable est exportée et que la clé commence par `ntn_` ou `secret_`.

### Page ou base non trouvée
**Cause :** L’intégration n’a pas accès à la page/base.  
**Solution :** Sur la page ou la base dans Notion, "..." → "Connecter à" → votre intégration.

### Requête ou filtre incorrect
**Solution :** Utilisez `notion-cli help` et la doc des filtres. Pour des requêtes complexes, utilisez le mode interactif : `notion-cli db query <DB_ID>` sans arguments.

---

**Auteur** : OpenClaw Skill  
Pour la référence complète des commandes et de l’API, consultez **SKILL.md**.
