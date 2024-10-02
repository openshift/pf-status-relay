# LACP Status Monitor and Relay for SR-IOV Interfaces

## Introduction
This application is designed for use with SR-IOV (Single Root I/O Virtualization) interfaces.
Its primary function is to monitor the Link Aggregation Control Protocol (LACP) status of Physical Functions (PFs) and relay this status by adjusting the link state of associated Virtual Functions (VFs).

## Purpose
In a typical scenario, a pod contains two VFs from different NICs, forming an active-standby bond (DPDK bond or Linux bond) with these VFs as slaves.
In the event of a switch connected to the PF with the active VF crashing, the VF remains active as the link carrier remains up. This results in the pod losing connectivity, as the standby VF does not become active. 

This application addresses this issue by adjusting the link state of the VF when the LACP is down.
LACP serves as a critical L2 failure detection mechanism on a single interface (PF).

## How it works
The application monitors the LACP status of the PFs and adjusts the link state of the VFs based on the LACP status.

When the LACP flags are not "Distributing", "Collecting", "Synchronization", and "Aggregation" on both LACP partner and actor, the application changes the link state of VFs whose current state is "auto" to "down". Subsequently, when the LACP flags return to the expected state (indicating a reconnection), the application changes the link state of VFs whose current link state is "down" to "auto".

For proper functionality, there must be a Linux bond for each PF that will be monitored (bond with a single slave), and the bond mode must be set to 802.3ad. If these conditions are not met, the application will not monitor or relay the LACP state. Additionally, LACP fast rate is expected to be used.

## Configuration
The application is configured using the following environment variables:
- `PF_STATUS_RELAY_INTERFACES`: A comma separated list of interfaces to monitor (i.e. "eth0,eth1").
- `PF_STATUS_RELAY_POLLING_INTERVAL`: The polling interval in milliseconds at which the application checks the LACP status. The default value is 1000 milliseconds.

## Usage

### Prerequisites

- SR-IOV capable network interface card
- Properly configured LACP, SR-IOV settings, and Linux bond mode (802.3ad)
- A Kubernetes cluster up and running.
- `kubectl` installed and configured to interact with your Kubernetes cluster.

### Installation

This application can be installed in a Kubernetes cluster using a DaemonSet.

#### Steps

1. **Build container image**

   First, you need to build the container image for the PF Status Relay application.

   ```bash
   make image-build
   ```

2. **Create a DaemonSet**

   Finally, you need to create a DaemonSet that runs the PF Status Relay application on each node in your cluster. Here's an example:

   ```yaml
   kind: DaemonSet
   apiVersion: apps/v1
   metadata:
     name: pf-status-relay-daemon
     namespace: default
   spec:
     selector:
       matchLabels:
         app: pf-status-relay
     template:
       metadata:
         labels:
           app: pf-status-relay
       spec:
         hostNetwork: true
         hostPID: true
         nodeSelector:
           kubernetes.io/os: linux
           node-role.kubernetes.io/worker: ""
         containers:
         - name: pf-status-relay
           image: localhost:5000/pf-status-relay
           env:
           - name: PF_STATUS_RELAY_INTERFACES
             value: "eno12409,ens6f0np0,ens6f1np1"
           - name: PF_STATUS_RELAY_POLLING_INTERVAL
             value: "500"
           securityContext:
             privileged: true
   ```

### Running the application
The following logs show how the link state of VFs is adjusted when LACPDU messages are intentionally blocked and unblocked on `ens6f0np0`.

```text
[root@cnfdr28-installer ~]# oc logs -f -n default pf-status-relay-daemon-gcp5w 
{"time":"2024-04-11T14:42:18.27643209Z","level":"INFO","msg":"Starting application"}
{"time":"2024-04-11T14:42:18.278692289Z","level":"INFO","msg":"pf is ready","interface":"eno12409"}
{"time":"2024-04-11T14:42:18.278752729Z","level":"INFO","msg":"pf is ready","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:18.278796494Z","level":"INFO","msg":"pf is ready","interface":"ens6f1np1"}
{"time":"2024-04-11T14:42:18.779615931Z","level":"INFO","msg":"lacp is up","interface":"eno12409"}
{"time":"2024-04-11T14:42:18.780551092Z","level":"INFO","msg":"lacp is up","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:18.781445769Z","level":"INFO","msg":"lacp is up","interface":"ens6f1np1"}
{"time":"2024-04-11T14:42:59.280236361Z","level":"INFO","msg":"lacp is down","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:59.28169901Z","level":"INFO","msg":"vf link state was set","id":0,"state":"disable","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:59.283745766Z","level":"INFO","msg":"vf link state was set","id":1,"state":"disable","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:59.285930289Z","level":"INFO","msg":"vf link state was set","id":2,"state":"disable","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:59.288096054Z","level":"INFO","msg":"vf link state was set","id":3,"state":"disable","interface":"ens6f0np0"}
{"time":"2024-04-11T14:42:59.289469469Z","level":"INFO","msg":"vf link state was set","id":4,"state":"disable","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.780785003Z","level":"INFO","msg":"lacp is up","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.782278877Z","level":"INFO","msg":"vf link state was set","id":0,"state":"auto","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.7838518Z","level":"INFO","msg":"vf link state was set","id":1,"state":"auto","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.786163397Z","level":"INFO","msg":"vf link state was set","id":2,"state":"auto","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.79731161Z","level":"INFO","msg":"vf link state was set","id":3,"state":"auto","interface":"ens6f0np0"}
{"time":"2024-04-11T14:44:38.799904792Z","level":"INFO","msg":"vf link state was set","id":4,"state":"auto","interface":"ens6f0np0"}
```

## Contributing
Contributions to this project are welcomed! If you encounter any issues or have suggestions for improvements, please feel free to submit a pull request or open an issue on GitHub.

## License
Please refer to the LICENSE file for more details.
