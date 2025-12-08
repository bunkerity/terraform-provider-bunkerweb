# Plan de Test Complet pour le Provider Terraform BunkerWeb

Ce répertoire contient un plan Terraform exhaustif pour valider toutes les fonctionnalités du provider BunkerWeb.

## 🎯 Objectif

Le fichier `comprehensive-test.tf` teste **toutes** les ressources, data sources, fonctions et ressources éphémères du provider :

### ✅ Data Sources (6 tests)
1. **`bunkerweb_global_config`** - Lecture de la configuration globale
2. **`bunkerweb_plugins`** - Liste des plugins disponibles
3. **`bunkerweb_jobs`** - Liste des jobs configurés
4. **`bunkerweb_cache`** - Éléments en cache
5. **`bunkerweb_service`** - Lecture d'un service spécifique
6. **`bunkerweb_configs`** - Liste de toutes les configurations

### ✅ Resources (7 tests)
7. **`bunkerweb_instance`** - Création d'une instance
8. **`bunkerweb_service`** (simple) - Service basique
9. **`bunkerweb_service`** (complexe) - Service avec configuration avancée
10. **`bunkerweb_global_config_setting`** - Paramètre global
11. **`bunkerweb_config`** - Configuration personnalisée
12. **`bunkerweb_ban`** - Bannissement d'IP
13. **`bunkerweb_plugin`** - Upload de plugin

### ✅ Functions (1 test)
14. **`provider::bunkerweb::service_identifier`** - Normalisation des noms de service

### ✅ Ephemeral Resources (8 tests)
15. **`bunkerweb_service_snapshot`** - Snapshot de service
16. **`bunkerweb_instance_action`** - Action sur instance (reload)
17. **`bunkerweb_service_convert`** - Conversion draft ↔ online
18. **`bunkerweb_config_upload`** - Upload de configuration
19. **`bunkerweb_config_upload_update`** - Mise à jour de configuration
20. **`bunkerweb_config_bulk_delete`** - Suppression en masse
21. **`bunkerweb_run_jobs`** - Exécution de jobs
22. **`bunkerweb_ban_bulk`** - Ban en masse

## 📋 Prérequis

1. **Instance BunkerWeb en cours d'exécution** avec l'API activée
2. **Token d'authentification** valide
3. **Provider compilé** et installé localement (voir le README principal)

## 🚀 Utilisation

### 1. Configuration de l'endpoint et du token

Éditez `comprehensive-test.tf` et modifiez les valeurs du provider :

```terraform
provider "bunkerweb" {
  api_endpoint = "http://127.0.0.1:8888/api"    # <- Votre endpoint
  api_token    = "YWRtaW46Y2hhbmdlTWUxMjMhCg==" # <- Votre token
}
```

### 2. Initialisation

```bash
terraform init
```

### 3. Validation du plan

```bash
terraform plan
```

Cette commande va :
- Lire toutes les data sources
- Planifier la création de toutes les ressources
- Exécuter les ressources éphémères
- Évaluer les fonctions

### 4. Application (optionnel)

```bash
terraform apply
```

⚠️ **Attention** : Cela va réellement créer des ressources sur votre instance BunkerWeb !

### 5. Destruction

Pour nettoyer après les tests :

```bash
terraform destroy
```

## 📊 Outputs de Validation

Le plan génère un output `test_summary` qui liste tous les tests effectués :

```hcl
output "test_summary" {
  value = {
    data_sources = {
      global_config = "✓ Testé"
      plugins       = "✓ Testé"
      jobs          = "✓ Testé"
      cache         = "✓ Testé"
      service       = "✓ Testé"
      configs       = "✓ Testé"
    }
    resources = {
      instance              = "✓ Créé"
      service_simple        = "✓ Créé"
      service_complex       = "✓ Créé"
      global_config_setting = "✓ Créé"
      config                = "✓ Créé"
      ban                   = "✓ Créé"
      plugin                = "✓ Créé"
    }
    functions = {
      service_identifier = "✓ Testé"
    }
    ephemeral_resources = {
      service_snapshot     = "✓ Testé"
      instance_action      = "✓ Testé"
      service_convert      = "✓ Testé"
      config_upload        = "✓ Testé"
      config_upload_update = "✓ Testé"
      config_bulk_delete   = "✓ Testé"
      run_jobs             = "✓ Testé"
      ban_bulk             = "✓ Testé"
    }
  }
}
```

## 🔧 Tests Ciblés

Si vous souhaitez tester uniquement certaines fonctionnalités :

### Tester uniquement les data sources
```bash
terraform plan -target=data.bunkerweb_global_config.test \
               -target=data.bunkerweb_plugins.test \
               -target=data.bunkerweb_jobs.test \
               -target=data.bunkerweb_cache.test
```

### Tester uniquement les resources
```bash
terraform plan -target=bunkerweb_instance.test_instance \
               -target=bunkerweb_service.test_app \
               -target=bunkerweb_service.test_api
```

### Tester uniquement les ressources éphémères
```bash
terraform plan -target=ephemeral.bunkerweb_service_snapshot.test_snapshot \
               -target=ephemeral.bunkerweb_instance_action.reload_test
```

## 🐛 Dépannage

### Erreur 403 Forbidden
Vérifiez que votre token API est correct et que l'authentification fonctionne :
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://127.0.0.1:8888/api/config/global
```

### Erreur de connexion
Vérifiez que l'API BunkerWeb est accessible :
```bash
curl http://127.0.0.1:8888/api/health
```

### Le provider n'est pas trouvé
Assurez-vous que le provider est correctement installé :
```bash
ls -la ~/.terraform.d/plugins/local/bunkerity/bunkerweb/0.0.1/linux_amd64/
```

### Erreur sur les ressources éphémères
Les ressources éphémères nécessitent Terraform >= 1.10. Vérifiez votre version :
```bash
terraform version
```

## 📝 Notes

- **Le test du plugin** nécessite un fichier `test-plugin.zip` qui a été créé automatiquement
- **Certains tests dépendent d'autres** (par exemple, `bunkerweb_config_upload_update` dépend de `bunkerweb_config_upload`)
- **Les ressources éphémères** s'exécutent pendant le `plan` et l'`apply`, mais ne persistent pas
- **Les outputs sensibles** (comme les résultats d'actions) sont marqués comme `sensitive = true`

## 🎓 Apprentissage

Ce plan de test sert également de **documentation par l'exemple** :
- Consultez les différentes ressources pour voir comment les utiliser
- Les commentaires expliquent chaque test
- Les outputs montrent comment extraire les données

## ⚡ Workflow de Développement

Pour tester rapidement après une modification du provider :

```bash
# 1. Recompiler le provider
cd /home/neus/dev/terraform-provider-bunkerweb
go build -o terraform-provider-bunkerweb

# 2. Le réinstaller
cp terraform-provider-bunkerweb ~/.terraform.d/plugins/local/bunkerity/bunkerweb/0.0.1/linux_amd64/terraform-provider-bunkerweb_v0.0.1

# 3. Nettoyer et retester
cd test-local
rm -rf .terraform .terraform.lock.hcl
terraform init
terraform plan
```

## 📚 Ressources

- [Documentation du Provider](../docs/)
- [Exemples](../examples/)
- [BunkerWeb Documentation](https://docs.bunkerweb.io/)
