# Issue #182: Authorization Sample Selection Algorithm

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
| Created | 2018-12-21 |
| Closed | N/A |
| Labels |  |
| Comments | 4 |

---

## Original Post

**Author:** @jagerman

https://github.com/graft-project/graft-ng/wiki/%5BDesign%5D-Authorization-Sample-Selection-Algorithm comments on the design of the supernode sample selection.  I have some comments/questions about the algorithm.

Most importantly, I have to ask: why *this* approach instead of some other approach?

I see some downsides that I'll get into, but this RFC (and the others) feel like they are simply describing what *is* being done rather than *why* it was chosen or is needed.  I can guess some of that, of course, but it would be quite valuable to have it written down why this aspect of the design was chosen to be the way it is.

What the algorithm describes is effectively uniform random sampling done in a deterministic way via a recent block hash and supernode public keys (whether the wallet public keys via the wallet address, or using a separate SN-specific public key as I suggest in https://github.com/graft-project/graft-ng/issues/176#issuecomment-446060076 doesn't really matter).

The big problem I see with this approach is this:

### Uniform random sampling leads to an enormously variable distribution of SN rewards.

Assuming a (long run) 50% supernode lock-in, with about 50% of the that going into T1 supernodes, we get somewhere around 9000 T1 supernodes expected on the network (once near maximum supply).

Thus, with this pure random selection formula, each T1 supernode would have a probability of `1 - (8999/9000)^2` (approximately 0.000222) of being selected in any block.

This in turn implies that there is only about a 14.7% chance of getting selected into the auth sample for at least one block in a day, and only a 67.4% chance of getting at least one auth sample entry in a week.

If your SN is online for 2 weeks, you still have slightly more than 10% chance of never being in the auth sample, and a 3.5% chance of never being in the auth sample after having your SN up for 3 weeks.

When considering get into the auth sample at least twice, the numbers are worse:
- 1.1% chance of getting 2+ auth samples in a day
- 30% chance of getting 2+ auth samples in a week
- 65.5% chance of getting 2+ auth samples in 2 weeks
- 95% chance of getting 2+ auth samples in a month

When you also consider the exponential distribution of block times, things look worse still because of the distribution of block times:

- 1.4% get less than 15 seconds of auth sample time per month
- 2.0% get between 15 and 60 seconds of auth sample time per month
- 3.9% get [1,2) minutes/month
- 5.1% get [2,3) minutes/month
- 6.0% get [3,4) minutes/month
- 6.6% get [4,5) minutes/month
- 7.0%, 7.0%, 6.9%, 6.6%, 6.2% get [5,6), [6,7), [7,8), [8,9), [9,10) minutes/month
- 5.7, 5.2, 4.7, 4.0, 3.6, 3.1, 2.6, 2.2, 1.9, 1.6% for [10,11) through [19,20)
- 5.9% get 20-30 minutes of auth time per month
- 0.6% get more than 30 minutes of auth time per month

If we then consider RTA earnings, the distribution becomes considerably more unequal still because of variation in the timing and amounts being spent.  The above represents a "best case" distribution where RTA payment amounts are constant, very frequent, and perfectly spread out over time.

I've deliberately chosen a 30-day timescale above because I believe that it is about as far as one can reasonable go while thinking that rewards will "average out."  As you can see above, though, they aren't averaging out in a reasonable time frame: even if RTA traffic was perfectly spread over time and for a constant amount, we have the top 10% of tier-1 SNs (ranking by auth sample time) earning seven times what the bottom 10% earns.

This sort of risk in reward distribution seems undesirable for potential SN operators and is likely to create a strong motivation for SN pooling--thus inducing centralization on the SN side of the network in the same way we have centralization currently among mining pool operators.

In Dash there is some randomness to MN selection, but it is strongly biased towards being a much fairer distribution: there is a random selection only from MNs that have not been one of the last 90% of MNs to earn a reward.  Unlike Graft, the reward is simply a portion of the block reward, so there is no extra time-dependent or transaction volume-dependent components to further spread out the distribution.  Loki is similar, but perfectly fair: SNs enter a queue and receive a payment when they reach the top.

One key distinction of Graft compared to both Dash and Loki, however, is that MN/SN sample selection in Dash/Loki is completely independent of MN/SN rewards.  In Loki, for example, there are performance metrics that a SN must satisfy or risk being deregistered (and thus losing rewards until the stake expires).  Dash, similarly, requires that MNs participate in network operations to stay active, foregoing any reward potential if they fail a network test and become inactive.

Neither of these are directly applicable to Graft, given the percentage nature of fees, but I feel that given the highly erratic nature of SN rewards that I laid out above this needs to be addressed.  Either a change to improve the fairness of SN rewards, or at least a solid explanation of why a fairer distribution of earnings isn't feasible.

Just to throw out a couple of ideas for discussion:

- have 5 queues (one queue for each tier plus a proxy SN queue).  Require that 0.5% of all RTA payments be burned, then remint some fraction (say 0.1%) of all outstanding burnt, non-reminted fees in each block and send an equal portion to the SN at top of each queue, returning that SN to the bottom of its queue.  Use network-assessed performance requirements to deregister (via a quorum) any SN with poor performance.

- Use 5 queues, as above, but just drop the RTA fee entirely and instead award SNs a constant fraction of the block reward (say 50%), combined with a meaningful tail emission (this could be one that declines over time until it hits a fixed level, or just a switch to an outright fixed emission level).

---

## Discussion Thread

### Comment by @Fez29

**Date:** 2018-12-21

A more reliably consistent/fairer reward distribution is desirable and makes sense. 

Potential SN operators would be much more likely to join the network if there was some sort of uniformity to rewards.

Especially if it encourages a more decentralised network and more SNs on the network. 

The least complicated ways of achieving  this should be seriously considered.

Regarding network assessed SN performance requirements - I do think this has value and could be used due to the fact that RTA is dependant on SNs response time and consistent up time especially if placed in a queue. As the Real Time Auth response time would obviously be a factor as it would be desired to be as short as possible or within some sort SLA. And SN performance requirements should reflect this but also take into account geographical differences to try promote an even distribution in location as well

---

### Comment by @Swericor

**Date:** 2018-12-22

Very interesting thoughts, I share your view that a more consistent reward system is needed.
I think however that delisting SNs due to poor performance is a bit harsh, especially if the que will be weeks long. Poor performing SNs could be shifted back one or a few steps in the que each time another SN has performed an auth and drops to the bottom of the que.

---

### Comment by @jagerman

**Date:** 2018-12-23

> Require that 0.5% of all RTA payments be burned, then remint some fraction

Thinking about this some more, this really won't fly while keeping RTA amounts secret.  (But on that note: a percentage-based fee for RTA payments doesn't allow for keeping RTA amounts secret in the first place).

---

### Comment by @Swericor

**Date:** 2018-12-26

Dropping a few steps in the que (for each newly processed block) would be a better incentive to get the SN online again asap. If you're immediately delisted, the offline-time doesn't really matter.

---

