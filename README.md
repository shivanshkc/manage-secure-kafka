# Secure Kafka Cluster - Ansible Deployment

Ansible playbooks to deploy a 3-node Apache Kafka cluster with TLS encryption and SASL authentication using Podman containers.

## Architecture

- **3 Kafka brokers** in KRaft mode (no ZooKeeper)
- **TLS encryption** for inter-broker and controller traffic
- **SASL/SCRAM-SHA-512** authentication for external clients
- **ACL-based authorization** for topic access control
- **Container runtime**: Podman

### Network Layout
- Port 9093: Controller listener (SSL, mutual TLS)
- Port 9094: Inter-broker listener (SSL, mutual TLS)
- Port 9095: Client listener (SASL_SSL)

## Prerequisites

- Ansible 2.9+
- Target machines running Debian/Ubuntu
- SSH access to all nodes
- Sudo privileges on target machines
- Local machine has `openssl` installed

## Quick Start

### 1. Configure Inventory

Inventory files for different environments live in the `inventories/` folder. Whichever environment
you intend to use, make sure to update `ansible.cfg` to point io it.

An inventory file looks like:
```yaml
all:
  hosts:
    kafka-machine-1:
      ansible_host: <IP_ADDRESS>
      ansible_user: <SSH_USER>
      ansible_ssh_private_key_file: <PATH_TO_SSH_KEY>
    kafka-machine-2:
      ansible_host: <IP_ADDRESS>
      ansible_user: <SSH_USER>
      ansible_ssh_private_key_file: <PATH_TO_SSH_KEY>
    kafka-machine-3:
      ansible_host: <IP_ADDRESS>
      ansible_user: <SSH_USER>
      ansible_ssh_private_key_file: <PATH_TO_SSH_KEY>
```

### 2. Configure Variables

**TLS Configuration:**
- `keystore_password`: Password for TLS keystores
- `local_tls_dir`: Local directory for generated certificates (default: `/tmp/kafka-tls`)

**Cluster Configuration:**
- `kafka_cluster_id`: Unique cluster ID (generate with `kafka-storage.sh random-uuid`)
- `kafka_num_partitions`: Default partitions per topic
- `kafka_replication_factor`: Replication factor
- `kafka_min_insync_replicas`: Minimum in-sync replicas for writes

**Topics & Users:**
- `kafka_topics`: List of topics to create
- `kafka_clients`: SASL users with passwords and ACL permissions

### 3. Deploy

Run all playbooks in order:
```bash
ansible-playbook playbooks/site.yml
```

Or run individually:
```bash
ansible-playbook playbooks/01-setup-podman.yml
ansible-playbook playbooks/02-setup-tls.yml
ansible-playbook playbooks/03-setup-kafka.yml
ansible-playbook playbooks/04-setup-kafka-users.yml
```

## Playbook Details

| Playbook | Purpose |
|----------|---------|
| `01-setup-podman.yml` | Installs Podman on all nodes |
| `02-setup-tls.yml` | Generates CA and broker certificates locally, distributes to nodes |
| `03-setup-kafka.yml` | Starts Kafka broker containers in KRaft mode |
| `04-setup-kafka-users.yml` | Creates topics, SASL users, and ACLs |

## Verification

Check cluster status from any broker:
```bash
# SSH into a broker
ssh <user>@<broker-ip>

# List topics
podman exec kafka-broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9094 \
  --command-config /tmp/command.properties \
  --list

# Describe cluster
podman exec kafka-broker /opt/kafka/bin/kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --print-details
```

Test client connection:
```bash
# Using kafkacat/kcat
kcat -b <broker-ip>:9095 \
  -X security.protocol=SASL_SSL \
  -X sasl.mechanism=SCRAM-SHA-512 \
  -X sasl.username=heimdall-observer \
  -X sasl.password=<password> \
  -X ssl.ca.location=/path/to/ca.crt \
  -L
```

## File Structure

```
.
├── ansible.cfg                         # Ansible configuration
├── inventories/
│   └── dev/
│       ├── hosts.yml                   # Inventory for dev environment
│       └── group_vars/
│           └── all.yml                 # Variables (cluster config, topics, users)
└── playbooks/
    ├── site.yml                        # Master playbook
    ├── 01-setup-podman.yml             # Install Podman
    ├── 02-setup-tls.yml                # Generate and distribute TLS certs
    ├── 03-setup-kafka.yml              # Deploy Kafka brokers
    └── 04-setup-kafka-users.yml        # Create topics, users, ACLs
```
