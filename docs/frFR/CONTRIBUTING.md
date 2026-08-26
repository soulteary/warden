# Guide de Contribution

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Merci de votre intérêt pour le projet Warden ! Nous accueillons toutes les formes de contributions.


## 📋 Table des Matières

- [Comment Contribuer](#comment-contribuer)
- [Configuration de l'Environnement de Développement](#configuration-de-lenvironnement-de-développement)
- [Standards de Code](#standards-de-code)
- [Standards de Commit](#standards-de-commit)
- [Processus de Pull Request](#processus-de-pull-request)
- [Rapports de Bugs et Demandes de Fonctionnalités](#rapports-de-bugs-et-demandes-de-fonctionnalités)

## 🚀 Comment Contribuer

Vous pouvez contribuer de plusieurs façons :

- **Signaler des Bugs**: Signaler des problèmes dans GitHub Issues
- **Suggérer des Fonctionnalités**: Proposer de nouvelles idées de fonctionnalités dans GitHub Issues
- **Soumettre du Code**: Soumettre des améliorations de code via des Pull Requests
- **Améliorer la Documentation**: Aider à améliorer la documentation du projet
- **Répondre aux Questions**: Aider les autres utilisateurs dans les Issues

Lors de la participation à ce projet, veuillez respecter tous les contributeurs, accepter les critiques constructives et vous concentrer sur ce qui est le mieux pour le projet.

## 🛠️ Configuration de l'Environnement de Développement

### Prérequis

- Go 1.26 ou supérieur
- Redis (pour les tests)
- Git

### Démarrage Rapide

```bash
# 1. Fork et cloner le projet
git clone https://github.com/your-username/warden.git
cd warden

# 2. Ajouter le dépôt en amont
git remote add upstream https://github.com/soulteary/warden.git

# 3. Installer les dépendances
go mod download

# 4. Exécuter les tests
go test ./...

# 5. Démarrer le service local (assurez-vous que Redis est en cours d'exécution)
go run .
```

## 📝 Standards de Code

Veuillez suivre ces standards de code :

1. **Suivre les Standards de Code Officiels de Go**: [Effective Go](https://go.dev/doc/effective_go)
2. **Formater le Code**: Exécuter `go fmt ./...`
3. **Vérification du Code**: Utiliser `golangci-lint` ou `go vet ./...`
4. **Écrire des Tests**: Les nouvelles fonctionnalités doivent inclure des tests
5. **Ajouter des Commentaires**: Les fonctions et types publics doivent avoir des commentaires de documentation
6. **Nommage des Constantes**: Toutes les constantes doivent utiliser le style `ALL_CAPS` (UPPER_SNAKE_CASE)

Pour des directives détaillées sur le style de code, veuillez vous référer à [CODE_STYLE.md](CODE_STYLE.md).

## 📦 Standards de Commit

### Format du Message de Commit

Nous utilisons le standard [Conventional Commits](https://www.conventionalcommits.org/) :

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: Nouvelle fonctionnalité
- `fix`: Correction de bug
- `docs`: Mise à jour de la documentation
- `style`: Ajustement du format de code (n'affecte pas l'exécution du code)
- `refactor`: Refactorisation du code
- `perf`: Optimisation des performances
- `test`: Relatif aux tests
- `chore`: Changements dans le processus de build ou les outils auxiliaires

## 🔄 Processus de Pull Request

### Créer une Pull Request

```bash
# 1. Créer une branche de fonctionnalité
git checkout -b feature/your-feature-name

# 2. Faire des modifications et commiter
git add .
git commit -m "feat: Ajouter une nouvelle fonctionnalité"

# 3. Synchroniser le code en amont
git fetch upstream
git rebase upstream/main

# 4. Pousser la branche et créer une PR
git push origin feature/your-feature-name
```

### Liste de Vérification de Pull Request

Avant de soumettre une Pull Request, assurez-vous que :

- [ ] Le code suit les standards de code du projet
- [ ] Tous les tests passent (`go test ./...`)
- [ ] Le code est formaté (`go fmt ./...`)
- [ ] Les tests nécessaires sont ajoutés
- [ ] La documentation associée est mise à jour
- [ ] Le message de commit suit les [Standards de Commit](#standards-de-commit)
- [ ] Le code passe les vérifications lint

Toutes les Pull Requests nécessitent une révision de code. Veuillez répondre rapidement aux commentaires de révision.

## 🐛 Rapports de Bugs et Demandes de Fonctionnalités

Avant de créer une Issue, veuillez rechercher les Issues existantes pour confirmer que le problème ou la fonctionnalité n'a pas été signalé.

## 🎯 Pour Commencer

Si vous souhaitez contribuer mais ne savez pas par où commencer, vous pouvez vous concentrer sur :

- Les Issues étiquetées `good first issue`
- Les Issues étiquetées `help wanted`
- Les commentaires `TODO` dans le code
- Les améliorations de documentation (corriger les fautes de frappe, améliorer la clarté, ajouter des exemples)

Si vous avez des questions, veuillez consulter les Issues et Pull Requests existantes, ou poser des questions dans les Issues pertinentes.

---

Merci encore de contribuer au projet Warden ! 🎉
