# Issue #1: Communication options

## Reception Score

| Score | Reason |
|-------|--------|
| **ACTIVE** | Open with discussion |

---

## Metadata

| Field | Value |
|-------|-------|
| State | OPEN |
| Author | @bitkis |
| Created | 2019-02-08 |
| Closed | N/A |
| Labels | Discussion |
| Comments | 6 |

---

## Original Post

**Author:** @bitkis

# Communication options


## Current state and motivation

Original P2P network is used for communication between supernodes. Announcements (messages of a special type) are periodically broadcast by every peer and are used for both keeping lists of active peers and building paths (tunnels) between the peers. Such approach induces a high value traffic in the network.

Yet another, less critical, issue is present in the current approach. Even though peers in the original P2P network have discoverable IPs, the complexity of IP discovery is exponential with respect to the number of peers in the network. However, any attempt to build a preferable path between 2 peers makes this complexity linear. 

Those issues were raised by *@jagerman* (see [#187](https://github.com/graft-project/graft-ng/issues/187)). The following document lists several approaches we considering, addressing the concerns.

When we first started working on issue, we were mainly focused on _Option 1_ since it would allow us to reduce the amount of traffic without making significant changes to current design. Options 3 and 4 were also under consideration. At the same time we started work on disqualification transactions design -- this mechanism means to be used in any case. Later, however, digging into _Options 3_ and _4_ brought us to _Option 2_, which we believe is the most optimal solution taking into account all practical considerations. 

**Publishing this document we would like to hear reaction of the community before making the final decision.**

Since there are still a few open issues, the estimates provided below are preliminary and may be changed if development scope needs to be extended.


## Optimization Options


### P2P broadcast optimization

We can reduce the amount of traffic (both keep-alive and data messages) during P2P broadcasts by

1.  Making it random for a peer to re-transmit a message further to the neighbors (same messages will not be re-transmitted by that peer but may be re-transmitted by a neighbor);
2.  Making it random for a peer to forward a message further to a particular neighbor (the message will be forwarded to a random subset of the neighbors);
3.  Reduce frequency of periodic broadcasts.

Reducing frequency of announcements, we, however, make both peer monitoring and building tunnels less robust.


### Disqualification transactions

Disqualification transaction is a special type of timed transactions in the blockchain, used to prevent a disqualified supernode from being selected to participate in an authorization sample. There are two mechanisms to issue a disqualification transaction:

1.  Every (second?) block randomly selected disqualification quorum "pings" a randomly selected supernodes from the set of supernodes with stack transactions in the blockchain and vote for disqualification of dead nodes.
2.  After an RTA transaction verification, authorization sample vote for disqualification of a supernode that didn't submit its vote or were late to vote during transaction verification.

Both mechanisms can be used either in conjunction or on their own.

## Development Paths

### Option 1: Keep current design and enhance it

*   Current design;
*   Optimized tunnel selection;
*   P2P broadcast optimization;
*   Announcement optimization
*   Disqualification transaction mechanism

#### Announcement optimization using Blockchain-based List

1.  Each supernode in an authorization sample checks if it's in the next (or few next) blockchain-based list(s). If included, it starts sending periodical announces over the network.
2.  While selecting an authorization sample, a supernode compares Blockchain-based list with Announcement List and selects only supernodes from which it receives the announces.
3.  Each supernode in an authorization sample checks if its blockchain-based list is active or the supernode is in the next blockchain-based list(s). If the blockchain-based list found inactive and the surernode is not in the next blockchain-based list(s), the supernode stops sending the announcement.

#### Tunnel selection

Currently, to build tunnels, graftnode selects only first three tunnels from announcement list for this supernode. However, at that moment, the list of peer connection can be different from the list which was at the moment of the receiving announce. In the case of increasing time delay between announcements, this situation becomes even more important. To optimize this, graftnode must select only tunnels which have active connections.

#### Pros

*   Easy to implement

#### Cons

*   Still suboptimal traffic (**not critical**)
*   Still linear complexity of IP lookups (**not critical**)

#### Open issues

*   Broadcast termination

#### Estimate

~2 weeks (testing included)

### Option 2: Implement Unstructured Distributed Hash Table (DHT)

*   Current design;
*   No announcements
*   P2P broadcast optimization;
*   Disqualification transaction mechanism.

1.  Upon a supernode joining the network, it retrieves the list of public identification keys from the blockchain (active supernodes), encrypts its IP using keys from a randomly selects subset, and broadcasts the encrypted IP over P2P network.
1.  Every few hours the supernode checks the selected supernodes are still active, and reselect inactive nodes. Then it repeats the broadcast procedure, described above.
1.  When sending a message, a supernode broadcasts it over P2P network. Broadcast is limited by a maximal number of hops. When the message reaches a node that knows recipient's IP, it's forwarded directly to the recipient.
1.  The recipient receives multiple copies of the same message, and should be able to handle this situation gracefully, with no noticeable performance degradation.

![dht-p2p](https://user-images.githubusercontent.com/36085298/52471459-caffa480-2b45-11e9-8503-f21c921d9a81.png)

On the figure above node A sends a message, addressed to node B. Nodes R retransmit the message issued by A. Nodes T terminate the broadcast, assuming 2 hops are allowed. DR nodes know IP of node B. 


#### Pros

*   Easy to implement
*   Almost optimal traffic
*   Fast communication between supernodes

#### Cons

*   Not quite optimal traffic

#### Open issues

*   There are several parameters that need to be selected properly.
*   Some math need to be done for proper estimations

#### Estimate

~ 2.5-3.5 weeks (testing included)

### Option 3: Supernode overlay/direct connections

We build a network overlay of supernodes, independent from P2P network. The overlay (or its subset) forms a DHT-like cluster. The DHT cluster can consists of full supernodes only. The DHT stores key-values pairs of supernode public identification keys and IPs. Both requests to join and queries are to be signed by private identification key and validated, upon entering DHT, against public identification key, retrieved from the blockchain. Peers in the supernode overlay communicate directly.

The disqualification transaction mechanism is used in this case as well.

![dht-query](https://user-images.githubusercontent.com/36085298/52471458-caffa480-2b45-11e9-86ec-b51319bcb5e8.png)

On the figure above supernode A, attempting to sends a message to supernode B, queries DHT first.


#### Pros

*   Optimal traffic
*   Fast communication between supernodes

#### Cons

*   All IPs are open to all valid supernodes
*   Requires extra development

#### Open issues

*   Distributed Hash Table (DHT) selection: Pastry seems to be most attractive right now.
*   DHT redundancy (most likely Pastry solves the issue)
*   Bootstrapping/entry point

#### Estimate

~3.5 weeks (testing included)


### Option 4: Supernode overlay/Hop over DHT

Again a network overlay of supernodes, independent from P2P network. The overlay forms a DHT-like cluster, where each node knows only small subset of the whole cluster. The DHT stores key-values pairs of supernode public identification keys and IPs. Unlike regular DHT that provides values in response to key-based queries, a sending peer passes a message itself to the DHT cluster. In case a cluster peer knows IP of the message's addressee, it forwards the message to the latter. Otherwise, the peer forwards the message to a known successor, according to the DHT algorithm.

Both requests to join and messages are to be signed by private identification key and validated, upon entering DHT, against public identification key, retrieved from the blockchain.

The DHT cluster can consist of full supernodes only. The number of hops, required for message delivery, does not exceed the number of successors.

![dht-messages](https://user-images.githubusercontent.com/36085298/52471457-caffa480-2b45-11e9-8d5e-f2e013abbe6a.png)

On the figure above supernode A sends a message to supernode B, passing it through DHT nodes.

#### Pros

*   Optimal traffic
*   Fast communication between supernodes

#### Cons

*   Requires extra development

#### Open issues

*   Distributed Hash Table (DHT) selection: Pastry seems to be most attractive right now.
*   DHT redundancy (most likely Pastry solves the issue)
*   Bootstrapping/entry point

#### Estimate
~4.5 weeks (testing included)

---

## Discussion Thread

### Comment by @jagerman

**Date:** 2019-02-08

One question missing from all of this is: *Why?*  Specifically, why is hiding supernode IPs particularly advantageous?

When hot wallets on the supernode were part of the design, the incentive for attack was obvious, but now that that has been eliminated, even if someone knows the IP of a supernode, there is little gain to be had from attacking it.

Without such secrecy, a much simpler alternative is:

# Option 5

Upon starting a supernode sends an announcement to the network containing (among other things) the IP and port on which it is reachable.  Ordinary nodes synchronize this list with each other.  Supernodes communicate directly.

---

### Comment by @bitkis

**Date:** 2019-02-08

> One question missing from all of this is: _Why?_ Specifically, why is hiding supernode IPs particularly advantageous?

To reduce probability of a DOS attack on an RTA auth sample

---

### Comment by @jagerman

**Date:** 2019-02-08

> To reduce probability of a DOS attack on an RTA auth sample

As I understand it, the auth sample is determined on the fly as needed and selected randomly based on a generated random value which can't be predicted; the timespan from when the auth sample is generated to when it is complete is measured in milliseconds.

---

### Comment by @jagerman

**Date:** 2019-02-08

Regarding Option 2: the only way to guarantee that the message from A to B actually reaches B is to make the hop limit equal to the diameter of the network graph.  To reuse your example from Option 2, here's the same graph but with some different edges:

![image](https://user-images.githubusercontent.com/4459524/52496327-69712180-2ba9-11e9-9474-910168643d9f.png)

You could increase it to a maximum of three, but then I could draw another counterexample where 3 doesn't work, and so on.  I could draw a connected network in your 15 node example where it requires 12 hops to reach any of the DRs (where I include B as a DR).

It seems that, since you have no guarantee at all of how connections are established, the only provably guaranteed value of T that will reach B is a value so is so absurdly large that it will reach every node on the network in the vast majority of cases.

---

### Comment by @bitkis

**Date:** 2019-02-08

> As I understand it, the auth sample is determined on the fly as needed and selected randomly based on a generated random value which can't be predicted; the timespan from when the auth sample is generated to when it is complete is measured in milliseconds.

An auth sample is selected from the list based on the block hash. So, DOS attack on that list can be an issue. Adding disqualification transactions makes such attack even more profitable (you can trigger disqualification of another's supernodes.)

---

### Comment by @bitkis

**Date:** 2019-02-08

> Regarding Option 2 [...]

There are two very relevant parameters here: hop limit and size of randomly selected subset of supernodes. We believe we can find an optimal combination of those. Also, I don't think it makes sense to talk about any guarantees, we rather talk about maximizing probabilities.

---

