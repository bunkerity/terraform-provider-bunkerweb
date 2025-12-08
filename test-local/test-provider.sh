#!/bin/bash
# Script d'aide pour tester le provider BunkerWeb localement

set -e

PROVIDER_DIR="/home/neus/dev/terraform-provider-bunkerweb"
TEST_DIR="$PROVIDER_DIR/test-local"
PLUGIN_DIR="$HOME/.terraform.d/plugins/local/bunkerity/bunkerweb/0.0.1/linux_amd64"
PROVIDER_BINARY="terraform-provider-bunkerweb_v0.0.1"

# Couleurs pour l'affichage
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

function print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

function print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

function print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

function print_error() {
    echo -e "${RED}✗ $1${NC}"
}

function build_provider() {
    print_header "1. Compilation du Provider"
    cd "$PROVIDER_DIR"
    
    if go build -o terraform-provider-bunkerweb; then
        print_success "Provider compilé avec succès"
    else
        print_error "Échec de la compilation"
        exit 1
    fi
}

function install_provider() {
    print_header "2. Installation du Provider"
    
    # Créer le répertoire si nécessaire
    mkdir -p "$PLUGIN_DIR"
    
    # Copier le binaire
    if cp "$PROVIDER_DIR/terraform-provider-bunkerweb" "$PLUGIN_DIR/$PROVIDER_BINARY"; then
        chmod +x "$PLUGIN_DIR/$PROVIDER_BINARY"
        print_success "Provider installé dans $PLUGIN_DIR"
    else
        print_error "Échec de l'installation"
        exit 1
    fi
}

function clean_terraform() {
    print_header "3. Nettoyage de Terraform"
    cd "$TEST_DIR"
    
    if [ -d ".terraform" ] || [ -f ".terraform.lock.hcl" ]; then
        rm -rf .terraform .terraform.lock.hcl
        print_success "Cache Terraform nettoyé"
    else
        print_warning "Pas de cache à nettoyer"
    fi
}

function init_terraform() {
    print_header "4. Initialisation de Terraform"
    cd "$TEST_DIR"
    
    if terraform init; then
        print_success "Terraform initialisé"
    else
        print_error "Échec de l'initialisation"
        exit 1
    fi
}

function validate_terraform() {
    print_header "5. Validation de la Configuration"
    cd "$TEST_DIR"
    
    if terraform validate; then
        print_success "Configuration Terraform valide"
    else
        print_error "Configuration invalide"
        exit 1
    fi
}

function plan_terraform() {
    print_header "6. Génération du Plan"
    cd "$TEST_DIR"
    
    echo -e "${YELLOW}Note: Cette étape peut échouer si l'instance BunkerWeb n'est pas accessible${NC}"
    echo ""
    
    if terraform plan -out=tfplan; then
        print_success "Plan généré avec succès"
        echo ""
        echo -e "${GREEN}Pour appliquer le plan: terraform apply tfplan${NC}"
        echo -e "${GREEN}Pour détruire les ressources: terraform destroy${NC}"
    else
        print_warning "Échec du plan (vérifiez la connexion à BunkerWeb)"
        return 1
    fi
}

function show_info() {
    print_header "Informations"
    echo "Provider directory: $PROVIDER_DIR"
    echo "Test directory: $TEST_DIR"
    echo "Plugin directory: $PLUGIN_DIR"
    echo ""
    echo "Fichiers de configuration:"
    ls -lh "$TEST_DIR"/*.tf 2>/dev/null || echo "Aucun fichier .tf trouvé"
    echo ""
    echo "Provider installé:"
    ls -lh "$PLUGIN_DIR/$PROVIDER_BINARY" 2>/dev/null || echo "Provider non installé"
}

function full_workflow() {
    build_provider
    install_provider
    clean_terraform
    init_terraform
    validate_terraform
    plan_terraform
}

# Menu principal
case "${1:-full}" in
    build)
        build_provider
        ;;
    install)
        install_provider
        ;;
    clean)
        clean_terraform
        ;;
    init)
        init_terraform
        ;;
    validate)
        validate_terraform
        ;;
    plan)
        plan_terraform
        ;;
    full)
        full_workflow
        ;;
    info)
        show_info
        ;;
    help|--help|-h)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  build     - Compiler uniquement le provider"
        echo "  install   - Installer uniquement le provider"
        echo "  clean     - Nettoyer le cache Terraform"
        echo "  init      - Initialiser Terraform"
        echo "  validate  - Valider la configuration"
        echo "  plan      - Générer le plan d'exécution"
        echo "  full      - Exécuter le workflow complet (défaut)"
        echo "  info      - Afficher les informations"
        echo "  help      - Afficher cette aide"
        echo ""
        echo "Exemples:"
        echo "  $0              # Workflow complet"
        echo "  $0 build        # Compiler uniquement"
        echo "  $0 plan         # Plan uniquement (après build+install)"
        ;;
    *)
        print_error "Commande inconnue: $1"
        echo "Utilisez '$0 help' pour voir les commandes disponibles"
        exit 1
        ;;
esac
