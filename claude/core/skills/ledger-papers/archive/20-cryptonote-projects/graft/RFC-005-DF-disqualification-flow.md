# Issue #2: Disqualification Flow

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
| Created | 2019-03-26 |
| Closed | N/A |
| Labels |  |
| Comments | 3 |

---

## Original Post

**Author:** @bitkis

Discussion placeholder for [[RFC-005-DF]-Disqualification-Flow](https://github.com/graft-project/DesignDocuments/blob/disqualification-flow/RFCs/%5BRFC-005-DF%5D-Disqualification-Flow.md)

---

## Discussion Thread

### Comment by @jagerman

**Date:** 2019-03-29

This is an algorithm description rather than a design document.

As far as the underlying design here goes, this seems overbuilt.  What is the point of a high level of complexity here?  Wouldn't it be far simpler to use a random quorum that votes on a random selection of supernodes, using a very simple rejection rule such as "no more than 3 missed authorizations in the last 720 blocks", and if the threshold is hit, submits *one* signed disqualification tx that kicks out the malfunctioning SN?  Why complex scores, extra data storage lists, and loads of magic numbers in calculations (such as: `0.5 + (DTBlockNumber - BDListBlockNumber) / (2 * (BlockHeight - BDListBlockNumber))`) of any benefit to the objective here?

Some particular things that jump out at me:

> - AAoS - Accumulated Age of stake - The value determines the reliability of the stake, based on the stake amount, number of blocks, passed after stake activation (as usual AoS) and average disqualification score (ADS), AoS = StakeAmount * StakeTxBlockNumber * (1 - ADS).

First, this is nonsense: there is no reason at all to suppose that T4 is 5 times as reliable as a T1, or that someone who stakes for a month at a time is (on average) 4 times as reliable as someone who stakes for a week at a time.

Second, this significantly undermining the integrity of the system, which relies on uniform random sampling.  By introducing controllable bias (i.e. use larger and longer stakes to greatly increase your chance of being selected) you weaken the security of the system.

> Gets first PBLSize bytes from the split block hash and selects PBLSize supernodes from it, using these one-byte numbers as indexes.

I honestly feel like I'm personally being trolled with this.  Using 1 byte of entropy for one random value is a *horrible* solution for anything that needs to be random other than something that needs exactly the range of one byte.  Please read over https://github.com/graft-project/GraftNetwork/pull/225 again.

---

### Comment by @bitkis

**Date:** 2019-04-04

@jagerman,

Let's hit on the common ground first:

> Wouldn't it be far simpler to use a random quorum that votes on a random selection of supernodes,

The quorum should be both random and verifiable, and all members of the quorum should be able to agree on the selection, correct?
 
> using a very simple rejection rule such as "no more than 3 missed authorizations in the last 720 blocks",

I assume you meant blockchain-based verification. So, do you suggest to go through all the RTA transactions in the last 720 blocks, reconstruct authorization samples for each of those, check if any of the randomly selected supernodes, mentioned above, missed participation in the corresponded samples? It doesn't look very simple. Also, what if an RTA transaction didn't make it to the black chain due to the malfunctioning supernode(s)?

> and if the threshold is hit, submits one signed disqualification tx that kicks out the malfunctioning SN?

Seems like you suggest skipping health checking ("pinging"), and kicking out the malfunctioning supernodes reactively, after harm has been already done. Is this correct?

> Why complex scores, extra data storage lists, and loads of magic numbers in calculations (such as: 0.5 + (DTBlockNumber - BDListBlockNumber) / (2 * (BlockHeight - BDListBlockNumber))) of any benefit to the objective here?

It was just an idea and we are to discuss it here. In general, we consider simplification of the process but the current concept attempts to make (1) assessment of auth sample work, since it can not always submit transaction (for example, auth sample does not get enough approvals) and we cannot check it using blockchain, (2) real-time network state estimation, "pinging" allows us to check health of supernodes in next Blockchain-based lists.

Current score schema is more complex than we'd like it to be but it allows us to take into consideration the age of disqualification transaction, since historical data cannot directly define the state of supernode but still provides important information of supernode's behavior.

> First, this is nonsense: there is no reason at all to suppose that T4 is 5e times as reliable as a T1, or that someone who stakes for a month at a time is (on average) 4 times as reliable as someone who stakes for a week at a time.

Yes, T4 is not more reliable as a T1, and in the process of building Blockchain-based list, different tiers form different lists  (see new revision of the document.) However, we still need verifiable order for supernodes and Age of stake is suitable for that.

> Second, this significantly undermining the integrity of the system, which relies on uniform random sampling. By introducing controllable bias (i.e. use larger and longer stakes to greatly increase your chance of being selected) you weaken the security of the system.

In our opinion, a long-term stake is more reliable for a sole reason: if the corresponding supernode misbehaved and got disqualified, the stake will stay locked for a longer time. So an owner of the longer  stake will be punished worse then an owner of a shorter one.

> I honestly feel like I'm personally being trolled with this. Using 1 byte of entropy for one random value is a horrible solution for anything that needs to be random other than something that needs exactly the range of one byte. Please read over graft-project/GraftNetwork#225 again.

Sorry, we missed to update the document properly. Updated now.


---

### Comment by @jagerman

**Date:** 2019-04-05

> The quorum should be both random and verifiable, and all members of the quorum should be able to agree on the selection, correct?

Yes.  This is why you seed a common RNG using common data such as the block hash at the height being considered.

> Seems like you suggest skipping health checking ("pinging"), and kicking out the malfunctioning supernodes reactively, after harm has been already done. Is this correct?

No, I suggest it in addition to a health check (but any such health check needs to be far more reliable than the current random mess where there is a non-negligible chance of false positive failures due to the randomness of announce forwarding).

A SN could be disqualified either because it did not stay up, or because it failed to complete authorizations.

> So, do you suggest to go through all the RTA transactions in the last 720 blocks, reconstruct authorization samples for each of those, check if any of the randomly selected supernodes, mentioned above, missed participation in the corresponded samples?

Yes.  Network rules must be enforced via concensus.  Right now you don't have any sample enforcement of RTA signatures in the design; this seems like a logical place for it.  Alternatively you could put it at the blockchain concensus layer (i.e. in graftnoded), and do active rejection of blocks with invalid samples, but that seems more complicated and would slow regular nodes down considerably.

> In our opinion, a long-term stake is more reliable for a sole reason: if the corresponding supernode misbehaved and got disqualified, the stake will stay locked for a longer time. So an owner of the longer stake will be punished worse then an owner of a shorter one.

So why allow shorter stakes *at all*?  If longer stakes are considered in your opinion to be more reliable, why would you ever want to allow shorter stakes (i.e. less reliable nodes) on the network?  Have fixed period (e.g. 30 day) more reliable stakes for everyone, or copy Loki's infinite stakes with long penalty periods (30 day continue lockup of stake) upon disqualification.

---

