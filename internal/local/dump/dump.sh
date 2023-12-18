#!/usr/bin/env bash

OUTPUT_ARCHIVE=""
INCLUDE_SENSITIVE=false
FULL_DISK_SCAN=false
DEBUG_MODE=false

show_help() {
    cat << EOF
Usage: $(basename "$0") [archive] [--include-sensitive] [--full-disk-scan] [--debug]

Description: This script collects various resources and information from a Kubernetes cluster with LWH deployed and generates an archive for further diagnostics

Arguments:
  archive   The path to the output archive file to be generated.

Options:
  -h, --help             Display this help message.
  --include-sensitive    Include sensitive data in the archive (e.g., values overrides). Use with caution.
  --full-disk-scan       Perform a full disk scan and include the detailed disk usage information in the archive.
  -v, --verbose          Increase verbosity to display debug information during collection phase.
EOF
}

# Parse command-line options
while [[ $# -gt 0 ]]; do
    key="$1"

    case $key in
        -h|--help)
            show_help
            exit 0
            ;;
        --include-sensitive)
            INCLUDE_SENSITIVE=true
            shift
            ;;
        --full-disk-scan)
            FULL_DISK_SCAN=true
            shift
            ;;
        -v|--verbose)
            DEBUG_MODE=true
            shift
            ;;
        -*)
            echo "Error: Unrecognized option: $key"
            show_help
            exit 1
            ;;
        *)
            OUTPUT_ARCHIVE="$1"
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z $OUTPUT_ARCHIVE ]]; then
    echo "Error: Output archive path is required."
    show_help
    exit 1
fi

# Check if script is run as root or with sudo
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root or with sudo"
   exit 1
fi

# Check required dependencies
command -v kubectl >/dev/null 2>&1 || { echo >&2 "kubectl is required, aborting."; exit 1; }
command -v k3s >/dev/null 2>&1 || { echo >&2 "k3s is required, aborting."; exit 1; }
command -v helm >/dev/null 2>&1 || { echo >&2 "helm is required, aborting."; exit 1; }
command -v tar >/dev/null 2>&1 || { echo >&2 "tar is required, aborting."; exit 1; }

RELEASE_NAMESPACE="home-weka-io"
RELEASE_NAME="wekahome"

# Function to collect resources for a given namespace
collect_namespace() {
    if [ "$#" -ne 1 ]; then
        echo "Namespace is required for collect_namespace"
        exit 1
    fi

    NAMESPACE="$1"

    mkdir -p "./resources/${NAMESPACE}"

    kubectl get all -n "${NAMESPACE}" > "./resources/${NAMESPACE}/resources.txt"
    kubectl describe pods -n "${NAMESPACE}" > "./resources/${NAMESPACE}/describe.txt"
    kubectl top pods --containers -n "${NAMESPACE}" > "./resources/${NAMESPACE}/metrics.txt"
    kubectl top nodes > ./resources/nodes-metrics.txt

    for POD in $(kubectl get pods -n "${NAMESPACE}" -o jsonpath='{.items[*].metadata.name}')
    do
        kubectl logs "${POD}"  -n "${NAMESPACE}" --previous --all-containers=true > "./resources/${NAMESPACE}/${POD}/previous-log.txt"
    done
}

# Function to collect NATS related information for a given namespace
collect_nats() {
    if [ "$#" -ne 1 ]; then
        echo "Namespace is required for collect_nats"
        exit 1
    fi

    mkdir -p ./nats

    NAMESPACE="$1"
    NATS_BOX=$(kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/component=nats-box -o jsonpath='{.items[*].metadata.name}')

    if ! timeout 5s kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats str ls >/dev/null; then
        echo "Failed connecting to NATS" >&2
        return 1
    fi

    kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats str report > ./nats/streams.txt
    kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats str ls >> ./nats/streams.txt
    kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats str ls -n | while IFS= read -r STREAM; do
        mkdir -p "./nats/${STREAM}"
        kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats con report "${STREAM}" > "./nats/${STREAM}/consumers.txt"
        kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats str info "${STREAM}" > "./nats/${STREAM}/info.txt"
        kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats con ls "${STREAM}" -n | while IFS= read -r CONSUMER; do
            mkdir -p "./nats/${STREAM}/${CONSUMER}"
            kubectl exec -n "${NAMESPACE}" "${NATS_BOX}" -- nats con info "${STREAM}" "${CONSUMER}" > "./nats/${STREAM}/${CONSUMER}/info.txt"
        done
    done

    return 0
}

collect_sensitive() {
    echo "Gathering Helm release info... (sensitive)"
    helm get all ${RELEASE_NAME} -n ${RELEASE_NAMESPACE} > ./helm-dump.txt
}

collect() {
    echo "Collecting cluster details..."
    kubectl get nodes > ./nodes-table.txt
    kubectl describe nodes > ./nodes.txt
    kubectl version >> ./cluster.txt

    k3s --version > ./k3s-version.txt
    k3s crictl info -o yaml > ./k3s-info.txt
    k3s crictl stats -o table > ./k3s-stats.txt
    k3s crictl statsp -o table > ./k3s-stats-pods.txt
    k3s ctr images check > ./k3s-images.txt

    systemctl status k3s > ./k3s-status.txt
    journalctl -u k3s -n 1000 > ./k3s-logs.txt

    echo "Collecting resources within namespaces..."
    kubectl cluster-info dump --all-namespaces --output-directory=./resources >/dev/null

    for NAMESPACE in $(kubectl get ns -o jsonpath='{.items[*].metadata.name}')
    do
        collect_namespace "${NAMESPACE}"
    done

    echo "Retrieving NATS details..."
    collect_nats ${RELEASE_NAMESPACE} >> ./errors.txt

    echo "Gathering Helm releases..."
    helm list -a -A > ./helm-charts.txt

    if [ "$INCLUDE_SENSITIVE" = true ]; then
        collect_sensitive
    fi

    echo "Inspecting CPU usage..."
    top -b -n 1 > ./cpu-top.txt
    ps aux --sort=-%cpu > ./cpu-ps.txt

    echo "Inspecting memory usage..."
    free -w -h > ./memory.txt

    echo "Inspecting disk space..."
    df -h > ./mounts.txt
    if [ "$FULL_DISK_SCAN" = true ]; then
        du -k / | sort -nr > ./disk-space.txt
    else
        du -k --max-depth 2 --total --exclude=/proc / | sort -nr > ./disk-space.txt
    fi

    echo "Copying syslog files..."
    cp -r /var/log ./syslog

    echo "Copying resolv.conf..."
    cp /etc/resolv.conf ./resolv.conf
}

main() {
    TEMPDIR=$(mktemp -d)
    mkdir -p "${TEMPDIR}/dump"
    pushd "${TEMPDIR}/dump" &>/dev/null || echo >&2 "Failed to switch to temp dir"

    if [ "$DEBUG_MODE" = true ]; then
        collect 2> >(tee ./errors.txt >&2)
    else
        collect 2> ./errors.txt
    fi

    popd > /dev/null 2>&1 || echo >&2 "Failed to switch back to initial dir"

    echo "Creating archive..."
    tar -C "${TEMPDIR}" -czf "${OUTPUT_ARCHIVE}" .
}

main "$@"
