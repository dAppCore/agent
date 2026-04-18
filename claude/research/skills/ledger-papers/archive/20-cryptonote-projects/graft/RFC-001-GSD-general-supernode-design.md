# Issue #187: [RFC 001 GSD] General Supernode Design

## Reception Score

| Score | Reason |
|-------|--------|
| **ACTIVE** | Open with discussion |

---

## Metadata

| Field | Value |
|-------|-------|
| State | OPEN |
| Author | @jagerman |
| Created | 2018-12-27 |
| Closed | N/A |
| Labels | RFC-draft |
| Comments | 4 |

---

## Original Post

**Author:** @jagerman

Some comments:

> The supernode charges the clients an optional fee for this activity.

Optional?

> Upon start, each supernode should be given a public wallet address that is used to collect service fees and may be a receiver of a stake transaction.

What is the point of this?  That receiving wallet is already included in the registration transaction on the blockchain; I don't see why the supernode needs to have a wallet (even just the wallet address) manually configured at all rather than just picking it up from the registration transaction.

> The supernode must regenerate the key pair per each stake renewal.

This is, as I have mentioned before, a very odd requirement.  It adds some (small) extra work on the part of the operator, and it would seem to make it impossible to verify when a SN is being renewed rather than newly registered (and thus not double-counted if it is both renewed and in the "overhang" period).  It also means that as soon as a SN stake is renewed (thus changing the key) any RTA requests that still use the old key simply won't be received by the SN in question.  In theory, you could make the SN keep both keys, but this raises the obvious question of: Why bother?  In #176 you wrote:

> You asked why we did not declare permanent supernode identification keypair. The main reason was that we didn't see any reason to make it permanent. The temporal keypair is enough for our goals and regeneration of this key won't create large overwork during stake renewal. And yes, the lifespan of this key pair will be equal to the stake period and during stake renewal supernode owner also need to update it. If someone wants to build a tracking system, they can do it anyway.

I carefully counted the number of benefits of mandatory regeneration provided in this description: 0.  So it has zero benefits and more than zero drawbacks.  So why is it here?

> Not storing any wallet related private information on supernode is a more secure approach, but it doesn't allow automatic re-staking.

Why not?  Other coins are able to implement automatic renewal without requiring a password-unprotected wallet or having the wallet on a service node; what part of the Graft design prevents Graft from doing what other coins have done?

> Stake transaction must include the following data:
> - the receiver of this transaction must be supernode's public wallet address;
> ...
> - tx_extra must contain supernode public wallet address;

This is a minor point, but it isn't entirely clear why this is required: you could simply include both a recipient wallet address and a reward recipient wallet to allow the possibility of wallet A to submit a stake with rewards going to wallet B, which seems like it could be useful.

> TRP determines the number of blocks during which supernode is allowed to participate in RTA validation even if it has no locked stake.  If during TRP supernode owner doesn't renew its stake transaction, the supernode will be removed from active supernode list and will not be able to participate in RTA validation.

And how, exactly, will you determine that the SN has been renewed since it won't have the old stake's pubkey anymore?

> The mechanism of periodic announcements has, therefore, a two-fold purpose:
> 1. make the best effort to deliver current status to all supernodes in the network without releasing the sender's IP to the whole network;

Verifying uptime is fine.  The design, however, of including incrementing hop counts makes it almost trivial to find the IP of any SN (or, at least, the graftnoded that the SN is connected to).

> 2. build reliable communication channels between any two active supernodes in the network without releasing IPs of the participants, while producing minimal traffic overhead. 

It may reduce traffic somewhat, but at the cost of a massive increase in traffic of frequent periodic traffic expenses that is almost certain to vastly eclipse any savings.  A simple back-of-the-envelope calculation:

    A = 2000 active service nodes (each of which a node will received an announce for)
    B = 1000 bytes per announce
    R = 1440 announces per day (= 1 announce per minute)
    N = 50 p2p connections typical for a mainnet node

    A * B * R * N = 144 GB of traffic per day both uploaded *and* downloaded just to transmit announces across the network.

And this isn't just incurred by supernodes, this is incurred by *all network nodes*.  Even if you decrease the announcement rate to 1 announce every 10 minutes you are still looking at 14GB/day of announcement traffic both uploaded and downloaded *which applies to ordinary network nodes*.

This is not a design that can be considered to incurs only "minimal traffic overhead".

> RTA validation participants may use encrypted messages.

"may"?

> ## Multiple Recipients Message Encryption

This whole feature seems rather pointless.  Multicast messages are going to have to be transmitted much more broadly than unicast messages: You can't just sent it along the best three paths, which you proposed for unicast messages, because each recipient is highly likely to have a completely different best three paths.  It doesn't seem like this multicast approach is going to save anything compared to simply sending 8 unicast messages (and then simplifying the code by dropping multicast support if there are no remaining cases for it).  There is potential for optimization here — you could use protocol pipelining to send all the unicast messages at once — the the proposed complexity added for encrypted multicast messages seems to have little benefit.

---

## Discussion Thread

### Comment by @bitkis

**Date:** 2019-01-04

> > Upon start, each supernode should be given a public wallet address that is used to collect service fees and may be a receiver of a stake transaction.

> What is the point of this? That receiving wallet is already included in the registration transaction on the blockchain; I don't see why the supernode needs to have a wallet (even just the wallet address) manually configured at all rather than just picking it up from the registration transaction.

The wallet address can be retrieved from StakeTx but the proposed approach unifies auth and proxy supernode handling.

> > The supernode must regenerate the key pair per each stake renewal.

> This is, as I have mentioned before, a very odd requirement. It adds some (small) extra work on the part of the operator, and it would seem to make it impossible to verify when a SN is being renewed rather than newly registered (and thus not double-counted if it is both renewed and in the "overhang" period). It also means that as soon as a SN stake is renewed (thus changing the key) any RTA requests that still use the old key simply won't be received by the SN in question. In theory, you could make the SN keep both keys, but this raises the obvious question of: Why bother?

Yes, we're considering both options.

> > Not storing any wallet related private information on supernode is a more secure approach, but it doesn't allow automatic re-staking.

> Why not? Other coins are able to implement automatic renewal without requiring a password-unprotected wallet or having the wallet on a service node; what part of the Graft design prevents Graft from doing what other coins have done?

Not sure what you meant here, unless you were talking about wallet side automation. What other coins have done that otherwise?

> > TRP determines the number of blocks during which supernode is allowed to participate in RTA validation even if it has no locked stake. If during TRP supernode owner doesn't renew its stake transaction, the supernode will be removed from active supernode list and will not be able to participate in RTA validation.

> And how, exactly, will you determine that the SN has been renewed since it won't have the old stake's pubkey anymore?

We don't really need to determine. If a supernode owner submits new StakeTx, the supernode starts to send announce with the new key, and old identification key just "expires".

Downtime problem during regular stake renewal can be fixed for the temporal key in the following way:
supernode, for which StakeTx unlocked, tracks it TRP, and if supernode owner renews stake transaction with a new identification key, supernode continues to send announces with the old identification key, until new StakeTx does not pass stake validation period (during this time this supernode knows both its identification keys.)

> > The mechanism of periodic announcements has, therefore, a two-fold purpose:
> > 1. make the best effort to deliver current status to all supernodes in the network without releasing the sender's IP to the whole network;

> Verifying uptime is fine. The design, however, of including incrementing hop counts makes it almost trivial to find the IP of any SN (or, at least, the graftnoded that the SN is connected to).

Well, not so trivial – for hop count h > 1, there are N^h possible peers in the h-neighborhood, where N is the "typical" number you mentioned bellow.

> > 2. build reliable communication channels between any two active supernodes in the network without releasing IPs of the participants, while producing minimal traffic overhead.
> It may reduce traffic somewhat, but at the cost of a massive increase in traffic of frequent periodic traffic expenses that is almost certain to vastly eclipse any savings. A simple back-of-the-envelope calculation:
>
> A = 2000 active service nodes (each of which a node will received an announce for)
> B = 1000 bytes per announce
> R = 1440 announces per day (= 1 announce per minute)
> N = 50 p2p connections typical for a mainnet node
>
> A * B * R * N = 144 GB of traffic per day both uploaded *and* downloaded just to transmit announces across the network.
>
> And this isn't just incurred by supernodes, this is incurred by all network nodes. Even if you decrease the announcement rate to 1 announce every 10 minutes you are still looking at 14GB/day of announcement traffic both uploaded and downloaded which applies to ordinary network nodes.

Well, in our estimate, B = ~ 200 bytes. Yes, decrease of the announcement rate is one possible optimization. Another one could be separation channel construction and state update parts, emitting the state changes only when they actually happen to a 1-hop neighbor.
Dropping the announcements at whole would leave us with no uptime verification and with need to broadcast all RTA traffic. The latter would produce much higher average load to the whole network, with no optimization options.
The only alternative we see here is building yet another p2p network, now between supernodes. Still, we'd have to fight the same issues, although on a relatively smaller domain. We want to avoid this path, at least for now, and have a fully working system, with may be a somewhat suboptimal traffic flow, fist.  

> This whole feature seems rather pointless. Multicast messages are going to have to be transmitted much more broadly than unicast messages: You can't just sent it along the best three paths, which you proposed for unicast messages, because each recipient is highly likely to have a completely different best three paths [...]

In our estimate, they're not so likely different.




---

### Comment by @jagerman

**Date:** 2019-01-04

> The wallet address can be retrieved from StakeTx but the proposed approach unifies auth and proxy supernode handling.

I don't understand how there is any benefit to doing this.  The auth SN simply needs an address, the proxy SN needs more than just an address.

> Not sure what you meant here, unless you were talking about wallet side automation.

I was.  I don't actually think that any automation that requires a hot wallet is a good idea, but if you're going to have it, it shouldn't be an unencrypted hot wallet (or, equivalently, an encrypted hot wallet with an password stored in a config file nearby) on the SN itself.

> Well, not so trivial – for hop count h > 1, there are N^h possible peers in the h-neighborhood, where N is the "typical" number you mentioned bellow.

If you didn't have the hop count included in the broadcast, this would indeed be true.  With with the hop count, the maximum number of nodes you would need to check to find the source is multiplicative, not exponential, because you wouldn't check the entire neighbourhood: you would only check the immediate connections and thus ignore all of those except one lowest-hop peer at each step.  The worst case is thus `Nh` connections, not `N^h`, and finding the source takes at most `h` announce cycles.  Someone with a bit of Monero-based coin experience could probably write code that could identify the source of any particular SN in a couple of hours.

Since this isn't actually offering SN originator IP anonymity, it isn't clear that there is any advantage at all; it would simplify a lot, greatly reduce the traffic, and not give up any secrecy if SN IP/port info could simply be public with SNs establishing direct connections.

> Downtime problem during regular stake renewal can be fixed for the temporal key in the following way: supernode, for which StakeTx unlocked, tracks it TRP, and if supernode owner renews stake transaction with a new identification key, supernode continues to send announces with the old identification key, until new StakeTx does not pass stake validation period (during this time this supernode knows both its identification keys.)

Sure, you can solve it this way, but this appears to be adding complexity in the design without any benefit at all: I'm still missing any explanation at all as to why key regeneration on renewal is an advantage.

> Well, in our estimate, B = ~ 200 bytes.

60 GB of traffic per day *just* for passing announces is still a couple of orders of magnitude too high.  This isn't optional traffic, either: every network node must pass it, not just nodes with supernodes attached.

There's also the fact that this announce mechanism *directly and independently* determines the set of active SNs in such a way that this list will often be inconsistent across nodes, as I have commented on in #185 .

The answer to *both* problems is to provide a strong incentive for SN operators to ensure that they stay online, and to unify online/offline information across the network.  You do the first one (incentive) by penalizing a node that misses performance targets.  You do the second one (unified information) by storing the information on active/inactive nodes in the blockchain.

So, for example, you could set a disqualification trigger at: haven't transmitted an hourly ping in >2 hours or have missed responding to >4 RTA requests.  If you hit either trigger, you get disqualified for 10 days (7200 blocks).  Then every period, a quorum of nodes would check a random subset of active supernodes for disqualification failures, and if a majority votes for disqualificiation, a disqualification tx would be submitted to the mempool.  As soon as that tx gets mined into the chain, all nodes immediately know the node is disqualified.  The SN list is the same everywhere, there's a strong incentive to ensure a reliable connection, pings can be done only hourly incurring minimal announce traffic, and you have total active SN consistency, thus allowing RTA auth sample verification.

---

### Comment by @bitkis

**Date:** 2019-01-07

> > Not sure what you meant here, unless you were talking about wallet side automation.

> I was. I don't actually think that any automation that requires a hot wallet is a good idea, but if you're going to have it, it shouldn't be an unencrypted hot wallet (or, equivalently, an encrypted hot wallet with an password stored in a config file nearby) on the SN itself.

Agree. And we actually went away from that.

> > Well, not so trivial – for hop count h > 1, there are N^h possible peers in the h-neighborhood, where N is the "typical" number you mentioned bellow.

> If you didn't have the hop count included in the broadcast, this would indeed be true. With with the hop count, the maximum number of nodes you would need to check to find the source is multiplicative, not exponential, because you wouldn't check the entire neighborhood: you would only check the immediate connections and thus ignore all of those except one lowest-hop peer at each step. The worst case is thus Nh connections, not N^h, and finding the source takes at most h announce cycles.

Sorry I don't see it this way. We might be off by 1 (depending how you count, it can be `N^{h-1}`) but it's still exponential:  you can check the immediate connections and ignore all of them except one lowest-hop peer _at the first step only_. You can't continue doing that unless you own the  whole h-neighborhood :)
No RPC API should/will provide the neighbor-hop map. And the IP anonymity is actually there.

> > Well, in our estimate, B = ~ 200 bytes.

> 60 GB of traffic per day just for passing announces is still a couple of orders of magnitude too high. This isn't optional traffic, either: every network node must pass it, not just nodes with supernodes attached.

We do believe the traffic can be significantly reduced. Anyway, the point is taken.

> So, for example, you could set a disqualification trigger at: haven't transmitted an hourly ping in >2 hours or have missed responding to >4 RTA requests. If you hit either trigger, you get disqualified for 10 days (7200 blocks). Then every period, a quorum of nodes would check a random subset of active supernodes for disqualification failures, and if a majority votes for disqualification, a disqualification tx would be submitted to the mempool. As soon as that tx gets mined into the chain, all nodes immediately know the node is disqualified. The SN list is the same everywhere, there's a strong incentive to ensure a reliable connection, pings can be done only hourly incurring minimal announce traffic, and you have total active SN consistency, thus allowing RTA auth sample verification.

Great idea, actually. We are looking at penalization right now, and the idea of the disqualification tx may be exactly the right one.

On the other hand I doubt the mechanism based on disqualification tx can be a primary guard in case of RTA: it's naturally slow. Yes, it lets us to punish a "bad" node but it doesn't help us to ensure _real time_ authorization on a short run. To me, we need both to penalize nodes that miss performance targets, _and_ to minimize possibility of RTA failure.

---

### Comment by @jagerman

**Date:** 2019-01-07

>> If you didn't have the hop count included in the broadcast, this would indeed be true. With with the hop count, the maximum number of nodes you would need to check to find the source is multiplicative, not exponential, because you wouldn't check the entire neighborhood: you would only check the immediate connections and thus ignore all of those except one lowest-hop peer at each step. The worst case is thus Nh connections, not N^h, and finding the source takes at most h announce cycles.

> Sorry I don't see it this way. We might be off by 1 (depending how you count, it can be N^{h-1}) but it's still exponential: you can check the immediate connections and ignore all of them except one lowest-hop peer at the first step only. You can't continue doing that unless you own the whole h-neighborhood :)
No RPC API should/will provide the neighbor-hop map. And the IP anonymity is actually there.

A remote node's peer list is literally the second thing exchanged (after the network id) when one node connects to a peer; this is a pretty fundamental part of the p2p communication layer.  So you can get the lowest-hop peer of your current peer list (call it A), close all your peer connections and open new connections to all A's recent peers.  Repeat `h` times; you'll now have the source node.

---

