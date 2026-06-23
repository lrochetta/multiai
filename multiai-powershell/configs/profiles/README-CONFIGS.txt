Chaque fichier .env correspond a un profil de lancement.

Tu peux modifier :
- les cles API
- les base URLs
- les model ids
- les timeouts
- les arguments de lancement

Champs metadata :
PROFILE_ID, SHORTCUT, TOOL, TOOL_LABEL, DISPLAY_NAME, DESCRIPTION, ORDER, COMMAND, ARGS, CLEAR_ENV, REQUIRED_SECRETS, SKIP_SECRET_CHECK

Toutes les autres lignes KEY=VALUE deviennent des variables d'environnement appliquees uniquement au process courant.

Gemini a ete retire du projet.
