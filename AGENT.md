# AGENT.md

## Project Overview

LACP Status Monitor and Relay for SR-IOV Interfaces. This application monitors
LACP (Link Aggregation Control Protocol) status on Physical Functions (PFs) and
relays status by adjusting link state of associated Virtual Functions (VFs).

**Purpose**: Prevents networking "black holes" where a virtual machine using a
VF believes its link is up, even though the underlying physical LACP bond has
failed. When LACP is down, VFs are set to "disable" state to force failover;
when LACP is up, VFs are set to "auto" state to enable traffic.

## Build and Development Commands

```bash
make build              # Build binary via hack/build.sh -> bin/pf-status-relay
make test-unit          # Run all unit tests with Ginkgo
make image-build        # Build container image (default: localhost:5000/pf-status-relay:latest)
make image-build IMAGE_REGISTRY=quay.io/<user> IMAGE_TAG=latest
```

Makefile variables: `IMAGE_REGISTRY` (default: localhost:5000), `IMAGE_NAME` (default: pf-status-relay), `IMAGE_TAG` (default: latest), `OCI_BIN` (default: docker)

## CI/CD

CI runs exclusively via Prow (`.ci-operator.yaml` + `openshift/release`).
GitHub Actions are disabled at org level. Prow jobs: images, unit, verify-deps.

## Virtual Testing with OVS/libvirt

`hack/virt-test.sh` creates virtual environment with Open vSwitch and KVM for testing.

**SR-IOV Emulation Support**: QEMU igb device (Intel 82576) supports SR-IOV emulation since QEMU 8.0/libvirt 9.3 (2023). Can create virtual VFs without physical hardware for testing VF link state control.

**Before running the script, check prerequisites and warn user about missing items:**

Check required packages:
```bash
for pkg in ovs-vsctl qemu-system-x86_64 virsh virt-install virt-customize virt-copy-in; do
  command -v $pkg >/dev/null 2>&1 || echo "Missing: $pkg"
done
```

Check versions for SR-IOV support (requires QEMU 8.0+, libvirt 9.3+):
```bash
qemu-system-x86_64 --version
virsh --version
```

Check group membership (use `groups` command - works with traditional and systemd-homed users):
```bash
for grp in libvirt kvm; do
  groups | grep -qw $grp || echo "User not in group: $grp"
done
```

Note: On Fedora, also check for `openvswitch` or `hugetlbfs` group for non-root OVS access. On Arch, OVS typically requires starting the systemd service.

If packages are missing, inform the user with appropriate install commands:
- Arch: `yay -S openvswitch qemu-full libvirt virt-install libguestfs guestfs-tools`
- Fedora: `sudo dnf install openvswitch qemu-kvm libvirt virt-install libguestfs-tools`

Note: On Arch, `virt-customize` is in the `guestfs-tools` package, while `virt-copy-in` is in `libguestfs`.

For group membership, detect if user is systemd-homed managed:
```bash
homectl inspect $USER >/dev/null 2>&1 && echo "systemd-homed" || echo "traditional"
```

Add groups (adjust list based on what's missing):
- Traditional users: `sudo usermod -aG libvirt,kvm $USER`
- systemd-homed users: `sudo homectl update $USER --member-of=$(homectl inspect $USER --json=short | jq -r '.memberOf[]' | paste -sd,),libvirt,kvm`

Note: After adding groups, logout/login required for changes to take effect.
