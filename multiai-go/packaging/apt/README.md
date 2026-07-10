# APT Repository for multiai

This directory contains the GPG signing key and configuration for the multiai APT repository (Ubuntu/Debian).

## Repository URL

```
deb [signed-by=/usr/share/keyrings/multiai-archive-keyring.gpg] https://lrochetta.github.io/multiai/apt stable main
```

## Quick Start for Users

```bash
# 1. Import the GPG key
sudo mkdir -p /usr/share/keyrings
sudo curl -fsSL https://lrochetta.github.io/multiai/apt/gpg-public.key \
  -o /usr/share/keyrings/multiai-archive-keyring.gpg

# 2. Add the repository
echo "deb [signed-by=/usr/share/keyrings/multiai-archive-keyring.gpg] https://lrochetta.github.io/multiai/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/multiai.list

# 3. Install
sudo apt update
sudo apt install multiai
```

## GPG Key Management

### Key details

- **UID**: multiai APT Repository `<laurent@rochetta.fr>`
- **Type**: RSA 4096-bit
- **Usage**: Signing only
- **Expiry**: 3 years from creation
- **Fingerprint**: `C386 EBA7 DD7F 742B 36C9 4BEE A6D9 99AB 129B 8351`

### Regenerating the key

If the key expires or needs rotation:

```bash
# Generate a new key (batch mode, no passphrase for CI)
gpg --batch --passphrase '' --quick-gen-key \
  "multiai APT Repository <laurent@rochetta.fr>" rsa4096 sign 3y

# Export the public key
gpg --armor --export "laurent@rochetta.fr" > packaging/apt/gpg-public.key

# Export the private key for GitHub Actions
gpg --armor --export-secret-keys "laurent@rochetta.fr" | \
  gh secret set APT_GPG_KEY

# Commit the new public key
git add packaging/apt/gpg-public.key
git commit -m "chore: rotate APT signing key"
```

### Setting the secret in GitHub

```bash
# Requires gh CLI and repo admin access
gpg --armor --export-secret-keys "laurent@rochetta.fr" | \
  gh secret set APT_GPG_KEY --repo lrochetta/multiai
```

## Repository Structure (gh-pages branch)

```
apt/
  gpg-public.key
  dists/
    stable/
      InRelease          # Clearsigned Release (for apt ≥ 0.6)
      Release            # Release metadata with checksums
      Release.gpg        # Detached GPG signature
      main/
        binary-amd64/
          Packages       # Package metadata (index)
          Packages.gz    # Compressed index
        binary-arm64/
          Packages
          Packages.gz
  pool/
    main/
      m/multiai/
        multiai_0.5.0_amd64.deb
        multiai_0.5.0_arm64.deb
```

## CI/CD Pipeline

The release workflow automatically:

1. Builds `.deb` packages via GoReleaser nfpm
2. Downloads them from the GitHub Release
3. Generates `Packages.gz`, `Release`, and GPG-signed `InRelease`
4. Pushes the result to the `gh-pages` branch

The script used is `scripts/generate-apt-repo.sh` in the repository root.

## Local Testing

To test the APT repository generation locally:

```bash
# Prerequisites
sudo apt install dpkg-dev apt-utils gnupg

# Generate the repo from .deb files
./scripts/generate-apt-repo.sh \
  --debs ./dist \
  --repo ./apt \
  --suite stable \
  --gpg-key "laurent@rochetta.fr"

# Serve locally for testing
cd ./apt
python3 -m http.server 8080

# Then on another terminal:
echo "deb [signed-by=/path/to/gpg-public.key] http://localhost:8080/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/multiai-test.list
sudo apt update
```
