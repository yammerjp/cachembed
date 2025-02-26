# Cachembed

Un proxy de cache léger pour les requêtes à l'API d'embedding d'OpenAI.

## Aperçu

Cachembed est un serveur proxy qui met en cache les résultats de l'API d'embedding d'OpenAI pour réduire les requêtes redondantes et minimiser les coûts. Il prend en charge SQLite (par défaut) et PostgreSQL comme backends de stockage.

## Fonctionnalités

- Met en cache les résultats d'embedding vers SQLite ou PostgreSQL
- Proxy les requêtes vers l'API d'OpenAI (https://api.openai.com/v1/embeddings par défaut)
- Prend en charge la validation de la clé API via un motif regex
- Restriction d'utilisation aux modèles d'embedding autorisés
- Prend en charge les migrations de base de données
- Configurable via des variables d'environnement

## Exigences

* Ruby 3.4.1 ou supérieur
* Rails 8.0.1 ou supérieur
* SQLite3 ou PostgreSQL

## Installation

Clonez le dépôt et installez les dépendances :

    git clone https://github.com/your-username/cachembed-rails
    cd cachembed-rails
    # si vous souhaitez utiliser PostgreSQL, exécutez :
    bundle install --with=postgresql
    # si vous souhaitez utiliser SQLite, exécutez :
    bundle install

## Configuration

Créez et migrez la base de données :

    bin/setup --skip=server

## Configuration

Configurez l'application en utilisant ces variables d'environnement :

| Variable d'Environnement     | Description                                      | Par défaut                             |
|------------------------------|--------------------------------------------------|---------------------------------------|
| CACHEMBED_UPSTREAM_URL       | Point de terminaison de l'API d'embedding d'OpenAI | https://api.openai.com/v1/embeddings  |
| CACHEMBED_ALLOWED_MODELS     | Liste de modèles autorisés, séparés par des virgules | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN    | Motif d'expression régulière pour la validation de la clé API | ^sk-[a-zA-Z0-9_-]+$                  |
| DATABASE_URL                 | Chaîne de connexion à la base de données          | Dépend de config/database.yml         |

## Utilisation

### Démarrage du Serveur

Environnement de développement :

    rails server

Environnement de production :

    RAILS_ENV=production rails server

### Points de Terminaison de l'API

Le serveur fournit le point de terminaison suivant :

- POST `/v1/embeddings`: Proxy les requêtes vers l'API d'embedding d'OpenAI avec mise en cache

Exemple de requête :

    curl -X POST http://localhost:3000/v1/embeddings \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer sk-your-api-key" \
      -d '{
        "input": "Votre texte ici",
        "model": "text-embedding-3-small"
      }'

## Licence

Licence MIT

## Contribution

Les pull requests sont les bienvenues ! Si vous trouvez un bug ou si vous souhaitez demander une fonctionnalité, veuillez ouvrir un problème.

## À faire

- Cache LRU (avec journaux de requêtes) et collecte des déchets pour les anciennes entrées de cache.