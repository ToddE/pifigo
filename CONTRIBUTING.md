# Contributing to pifigo

Thank you for your interest in contributing to `pifigo`! We welcome contributions from everyone, whether it's reporting bugs, suggesting new features, improving documentation, or submitting code changes. Your help makes `pifigo` better for everyone.

## Table of Contents
- [Contributing to pifigo](#contributing-to-pifigo)
  - [Table of Contents](#table-of-contents)
  - [1. Code of Conduct](#1-code-of-conduct)
  - [2. How Can I Contribute?](#2-how-can-i-contribute)
    - [Reporting Bugs](#reporting-bugs)
    - [Suggesting Enhancements](#suggesting-enhancements)
    - [Improving Documentation](#improving-documentation)
    - [Submitting Code Changes](#submitting-code-changes)
  - [3. Development Setup](#3-development-setup)
  - [4. Coding Guidelines](#4-coding-guidelines)
  - [5. Commit Messages](#5-commit-messages)
  - [6. Pull Request Process](#6-pull-request-process)
  - [7. Localization Support](#7-localization-support)
    - [How to Contribute a New Language:](#how-to-contribute-a-new-language)
  - [8. Licensing](#8-licensing)
  - [9. Code of Conduct](#9-code-of-conduct)
  - [10. Contact](#10-contact)

## 1. Code of Conduct

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md) (if you create one). By participating in this project, you agree to abide by its terms.

## 2. How Can I Contribute?

### Reporting Bugs

If you find a bug, please open an issue on the [GitHub Issues page](https://github.com/ToddE/pifigo/issues).
Before opening a new issue, please search existing issues to see if your problem has already been reported.

When reporting a bug, please include:
* A clear and concise description of the bug.
* Steps to reproduce the behavior.
* Expected behavior.
* Actual behavior observed.
* Your device's hardware (e.g., Raspberry Pi Zero 2 W, Orange Pi Zero 3) and OS (e.g., Raspberry Pi OS Lite 64-bit, Armbian Minimal).
* Logs (e.g., `sudo journalctl -u pifigo.service -f` output) if applicable.

### Suggesting Enhancements

We love new ideas! If you have a suggestion for an enhancement or a new feature, please open an issue on the [GitHub Issues page](https://github.com/ToddE/pifigo/issues).

When suggesting an enhancement, please describe:
* The problem you're trying to solve.
* The proposed solution or feature.
* How it would benefit `pifigo` users.

### Improving Documentation

Good documentation is crucial! You can contribute by:
* Suggesting improvements to the `README.md`.
* Correcting typos or unclear phrasing in existing documentation.
* Adding new documentation for features that are not well-covered.
* Opening a Pull Request with your proposed changes.

### Submitting Code Changes

If you'd like to contribute code, please follow the steps below. This is generally for fixing bugs, implementing new features, or improving existing functionality.

## 3. Development Setup

To get your development environment ready:

1.  **Fork the Repository:**
    Go to the [pifigo GitHub repository](https://github.com/ToddE/pifigo) and click the "Fork" button in the top right corner. This creates a copy of the repository under your GitHub account.
2.  **Clone Your Fork:**
    Clone your forked repository to your local machine:
    ```bash
    git clone [https://github.com/YOUR_GITHUB_USERNAME/pifigo.git](https://github.com/YOUR_GITHUB_USERNAME/pifigo.git)
    cd pifigo
    ```
3.  **Set Upstream Remote:**
    Add the original `pifigo` repository as an "upstream" remote:
    ```bash
    git remote add upstream [https://github.com/ToddE/pifigo.git](https://github.com/ToddE/pifigo.git)
    ```
    This allows you to easily fetch changes from the main project.
4.  **Install Go:**
    Ensure you have [Go 1.24 or newer](https://go.dev/dl/) installed on your development machine.
    * Verify: `go version`
5.  **Install Dependencies & Build:**
    Go modules will handle dependencies. To build for local testing and target architectures:
    ```bash
    go mod tidy             # Synchronize dependencies
    chmod +x build.sh       # Make the build script executable
    ./build.sh              # Build for all supported Pi architectures
    ```

## 4. Coding Guidelines

* **Go Formatting:** All Go code should be formatted using `gofmt`. Your IDE (VS Code with Go extension) usually does this automatically on save.
* **Linting:** We recommend running `golangci-lint` (or similar linters) to catch common issues.
* **Testing:** Write unit tests for new functionality and ensure existing tests pass.
    * Run tests: `go test ./...`
* **Clarity:** Write clear, concise, and well-commented code.

## 5. Commit Messages

Please follow a conventional commit message format (e.g., [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)). This helps with generating changelogs and understanding the purpose of commits.

Example:
<div style="border: 1px solid grey; padding: 10px;">
<p style="font-size: 0.85em; font-weight: 700; margin-bottom: 4px">Add a Title:<span style="color: red">*</span></b>
<div style="border: 2px solid darkblue; border-radius: 10px; padding: 6px; font-weight: 600;">
FEATURE: Improved Support for NetworkManager 
</div>
<p style="font-size: 0.85em; margin-top: 14px; margin-bottom: 4px; font-weight: 700; ">Add a Description:</b>
<div style="border: 1px solid grey; padding: 6px; font-size: 0.9em;">
This commit introduces NetworkManager implementation for Wi-Fi setup, allowing pifigo to adapt to systems using NM.<br/><br/>
</div>

## 6. Pull Request Process

1.  **Create a New Branch:**
    For each bug fix or feature, create a new branch from `main`:
    ```bash
    git checkout main
    git pull upstream main # Ensure your main branch is up-to-date
    git checkout -b feature/my-new-feature-name # Or bugfix/fix-issue-xyz
    ```
2.  **Make Your Changes:**
    Implement your changes, write tests, and ensure all existing tests pass.
3.  **Commit Your Changes:**
    Commit your changes with clear commit messages.
    ```bash
    git add .
    git commit -m "feat: implement X feature"
    ```
4.  **Push Your Branch:**
    ```bash
    git push origin feature/my-new-feature-name
    ```
5.  **Open a Pull Request (PR):**
    Go to the [pifigo GitHub repository](https://github.com/ToddE/pifigo) in your browser. GitHub will usually prompt you to open a PR from your newly pushed branch.
    * Provide a clear PR title summarizing the changes.
    * Write a detailed description explaining what your PR does, why it's needed, and how it was tested.
    * Reference any related issues (e.g., `Closes #123`).
6.  **Review Process:**
    Your PR will be reviewed by the project maintainers. Be prepared to address feedback and make further changes.
    *(You may need to sign a Contributor License Agreement (CLA) if [Your Name/Organization] requires one, but for simple MIT-licensed projects, this is less common for individual contributors unless specific policies in place.)*

## 7. Localization Support

**pifigo** aims to be a universally usable tool, and supporting multiple languages is a key part of that. We welcome contributions from translators!

### How to Contribute a New Language:

1.  **Find the Language Files:** All UI strings are externalized in TOML files located in the `lang/` directory of this repository (e.g., `lang/en.toml`, `lang/fr.toml`).
2.  **Choose Your Language:**
    * Select the ISO 639-1 code for your target language (e.g., `es` for Spanish, `de` for German, `uk` for Ukrainian, `zh` for Simplified Chinese, `pt` for Brazilian Portuguese, `it` for Italian).
    * If a file for your language already exists, great! You can suggest improvements.
3.  **Create/Copy a File:** If your language doesn't exist, copy an existing file (e.g., `en.toml`) to a new file named with your language code (e.g., `lang/es.toml`).
4.  **Translate Strings:** Go through each `key = "value"` pair and translate the `value` into your chosen language. **Do not change the `key` name.**
    * Pay attention to placeholders like `%s` in strings like `success_message_template`. These are replaced by dynamic values at runtime. Do not translate or remove the `%s`.
5.  **Test (Optional but Recommended):** If you can build `pifigo` locally, you can test your translation by setting `language = "your_lang_code"` in your `config.toml` and running the app.
6.  **Submit a Pull Request:** Open a Pull Request on GitHub. In your PR description, mention the language you've added or updated.

Thank you for helping make **pifigo** accessible to more users worldwide!

## 8. Licensing
By contributing to **pifigo**, you agree that your contributions will be licensed under the [MIT License](LICENSE.md).

## 9. Code of Conduct 

*This is a new project -- we're still working on our Code of Conduct.* 

**Short Version:** 
1. Be nice. 
2. Don't be a jerk. (see #1)
3. Do no harm. 

## 10. Contact

If you have any questions, feel free to open an issue or reach out to [project maintainers, e.g., @ToddE on GitHub].
