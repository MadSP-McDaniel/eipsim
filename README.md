# EIPSim

EIPSim is a flexible simulator for cloud IP address pools. It allows simulation using both synthetic client behaviors (drawn from random distributions) and real-world traces (e.g., from Google's `clusterdata-2019` dataset).

# Requirements/Dependencies

*Note:* This artifact includes traces via [Git LFS](https://git-lfs.com/), which must be installed for tests to work.

EIPSim can run on any recent amd64 or arm64 CPU. Go tests are run in parallel; we recommend 4GB of memory per vCPU (Our tests were performed on an AWS `m6i.4xlarge` with 16 vCPU and 64GB memory). If you have insufficient memory, you can change the number of parallel jobs using the `-parallel n` flag.

EIPSim requires Go 1.18 or later. Dependencies (tools for decompressing input data) can be installed using

    go get ./...

Figure generation code relies on Python 3 and Matplotlib. This can be installed with

    pip3 install matplotlib

EIPSim can also be run using the included `Dockerfile`:

    docker build -t eipsim .
    docker run -it eipsim bash

# Reproducing Paper Results

All eval results from the paper are packaged as Go tests. They can be run as follows:

    go test -v ./eval/...  -bench . -timeout 10h

All tests take roughly 64 vCPU-hours to run (highly parallelizable).

Results will be stored as `.jsonl` files in the `eval/figs` directory. All figures can then be produced using

    cd eval && python3 figs.py

# Implementation Details

While the the scale of cloud computing as astronomical, the allocation of IP addresses occurs and can be simulated independently. While the space of these addresses is still quite large (~16M) for the largest AWS cloud region, this is still within the realm of exact simulation. To this end, EIPSim simulates concrete tenant and cloud provider behavior at IP- and second-level granularity.

Within this architecture, *Agents* perform the behavior of tenants, either by simulating tenant behavior or by replaying allocation traces from a previous run of the simulator or from actual tenants. The simulator fulfills IP allocation requests from these agents by referring to implementations of an *IP pool policy*. Each agent has the ability to allocate IPs under multiple tenant IDs, and the simulator treats these allocations as though they come from different tenants. While processing allocations, the simulator records statistics on the lifecycle of addresses, associated latent configuration, and adversarial objectives. Importantly, while these results are aggregated across addresses and tenants, they are a product of granular simulation of each tenant IP allocation.

## Tenant Agents

EIPSim relies on tenant agents (See `agents` folder) to perform the allocations of tenants. At each time-step (1s) the simulator allows each agent to perform actions. Benign behaviors can be simulated by one of two agents:

* The *benign tenant agent* simulates the allocation behavior of tenants scaling cloud resources. For each tenant managed by the agent, and at each time-step, the agent checks if the tenant should allocate or release IP addresses, and passes these actions back to the simulator.
* The *file agent* allows loading of tenant behaviors from a time-series file. This file contains the timestamps and tenant IDs of each IP allocation and release from either a previous run of EIPSim or recorded from a live cloud environment.


The *adversarial agent* is a specialized agent designed to simulate and analyze the behavior of a single- or multi-tenant adversary. The adversarial agent performs allocations exactly as it would on a real system (except that allocation requests are passed to the simulator instead of a cloud provider), and proceeds in several steps:

1. The agent requests IP addresses from the provider up to some quota (the maximum number of IPs it will hold at once). It records previous tenants and latent configuration associated with these for analysis (in reality an adversary would listen for network traffic or search DNS databases to identify these).
2. The agent holds these IPs for a fixed duration. In EIPSim, this is accomplished by performing no action when called by the simulator during this time.
3. The agent releases IPs that have been held for the specified duration back to the pool.
4. In the case of a multi-tenant adversary, new IPs are allocated under new tenant IDs. After a maximum tenant ID is reached, the adversary loops back to the initial tenant, simulating an adversary with access to only a fixed number of tenants.

While the techniques employed by the adversary could be performed by any cloud customer, the adversarial agent has access to the internal data structures of the simulator to be able to record time-series data on the functioning of the pool. For instance, when the agent allocates IP addresses it can access the list of previous tenants associated with that address (as this is used by our analysis). 

## Allocation Policies

When the simulator receives a request for an IP address from a tenant, it forwards it to an allocation policy (See `policies` folder) for servicing. While the simulator tracks what IP addresses are in use at any time, it is ultimate up to the policy to determine which free IP address is allocated to a given tenant. The policy receives the tenant ID associated with each allocation, but is not told the agent performing the request, or if the tenant is adversarial. The policy must also service all requests, though it may return any free IP for a given request.

The policy contains data structures that can track the history of a given IP address. For instance, the Segmented policy tracks the most recent tenant ID for each IP, the cooldown time, and the average allocation durations of tenants. When a tenant requests an IP address, it heuristically samples available IPs that best conform to the policy based on this data.

## Extending the EIPSim Framework
EIPSim supports expansion to new policies, behaviors, and adversaries as academics and practitioners continue to study cloud IP allocation. EIPSim defines `interface`s between components, and new components can be added either as part of the EIPSim package, or within a separate program that uses EIPSim as a library. EIPSim provides convenience functions to ease in the development of new components: for example, our studied allocation policies were implemented in an average of 71 lines of code, and new parameter sweep tests can be built on top of EIPSim in around 70 lines of code. We expect that, by encouraging the development of new components on top of our framework, the community can reach a unified means to compare threat models and defenses. EIPSim also supports allocation traces collected by cloud providers through custom agents. Practitioners can directly read allocations as tuples of $(T, t_a, t_r)$ and use EIPSim to simulate adversarial and pool behavior.

# Paper Reference

```
@inproceedings{pauley_secure_2025,
  address   = {San Diego, CA},
  title     = {Secure {IP} {Address} {Allocation} at {Cloud} {Scale}},
  booktitle = {2025 {Network} and {Distributed} {Systems} {Security} {Symposium} ({NDSS})},
  publisher = {Internet Society},
  author    = {Pauley, Eric and Domico, Kyle and Hoak, Blaine and Sheatsley, Ryan and Burke, Quinn and Beugin, Yohan and Kirda, Engin and McDaniel, Patrick},
  month     = feb,
  year      = {2025}
}
```