# Issue #191: [RFC 003 RTVF] RTA Transaction Validation Flow

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
| Created | 2019-01-10 |
| Closed | N/A |
| Labels |  |
| Comments | 8 |

---

## Original Post

**Author:** @jagerman

Comments.  Two major, a few smaller issues.

# Privacy leakage.

This design leaks privacy to the PoS proxy, the auth sample, and the wallet proxy.  To quote from https://www.graft.network/2018/11/21/how-graft-is-similar-to-and-at-the-same-time-different-from-visa-and-other-payment-card-networks-part-2/

> This property is **absolute privacy** provided by GRAFT Network to both buyer and merchant. Unlike plastic cards and most cryptocurrencies, GRAFT’s sender address, recipient address, transaction amount, and transaction fee amount are invisible to everyone except for the sender and recipient themselves.

This design, however, does not accomplish that: the PoS proxy is able to identify all payments received by the PoS, and all SNs involved in the transaction see the amount sent (even if they can't see the recipient address).

A cryptocurrency that is only private as long as you have to trust a single party (the PoS proxy) is no longer a privacy coin.

But it gets worse: from the description in the RFC it is possible for various network participants other than the receiving and paying wallets to get "serialized payment data" which consists of "serialized payment data – list of purchased items, price and amount of each item, etc.".

So, to summarize the privacy leaks that seem to be here:

- the PoS proxy SN sees the recipient wallet address, the total amount, and individual items purchased including the amount of each item.
- auth sample SNs see the total amount including the amount received by the proxy PoS
- wallet proxy SN plus, apparently, *any* SN can get an itemized list of the transaction



# Other comments

- this design has no protection against a selfish mining double-spending attack.  Unlike a double-spending attack against an exchange, double-spending here does not have to reach any minimum number of confirmations; *and* can be timed (with a little effort) to not even require 51% of the network.  (I pointed this out just over two months ago in the public JIRA with details of how to carry out an attack *and a demo* but the issue has had no response).

(`4. Regular key image checking (double spent checking.)` does nothing against the above attack: the key image *isn't* spent on the network visible to the SNs until the private block is released.)

- The PoS <-> PoS proxy SN communication layer should be encrypted so that the PoS can verify it is talking to the expected party (since the PoS in this design has to be trusted with all RTA payment data).  This should require HTTPS (with certificate validation enabled), or something similar, both to encrypt the data against MITM snooping, but also importantly to avoid someone spoofing the PoS proxy connection to send false authorization updates back to the PoS.

> 10. Each supernode from auth sample and PoS Proxy Supernode ...

There is a huge amount of complexity added here for little apparent reason.  You set the success/failure conditions at 6/3 replies so that you have can have a consistent concensus among the SNs, which I understand, but you don't *need* this success/failure concensus when you have a single party that is in charge: the PoS proxy.

If you simply changed the rules so that the PoS proxy is always the one to distribute the block, you would simplify the traffic (SN auth sample results can be unicast to the PoS proxy, and the payment success can simply be a state variable that never needs to be broadcast over the network), but more importantly you would allow a 6/1 success/failure trigger without incurring any consistency problem.

> ii. Transaction considered to be rejected in the case at least 3 out of 8 auth sample members or PoS Proxy rejected it.

Allowing 2 failures is a recipe for fee cheating: hack your wallet to reduce two of the eight SN fees to zero (or just leave them out) in every transaction to give yourself a small rebate.

> iii. When any auth sample supernode or PoS Proxy Supernode gets in:

What happens if there are 5 successes, 2 failures, and one timeout?

> Graftnode that handles RTA transaction validates:
> i. Correctness of the selected auth sample;

Which is done how, exactly?  In particular, how much deviation from what it thinks is correct will it allow?  This needs to be specified.

> 12. Once the graftnode accepts the transaction, supernode, which submitted it to the cryptonode, broadcasts successful pay status over the network

Why is this needed at all?  Success can already been seen (and is already transmitted across the network) by the fact that the transaction enters the mempool.  Can't the wallet just check for that instead?


# This design is non-trustless!

This design puts far too much centralized control in the hands of the proxy SN.  The design here puts this single node as RTA transaction gatekeeper, with the possibility to lie to the PoS about transaction validity—a lie here could be deliberate, or could be because the proxy SN in use was hacked.  This is not how a decentralized cryptocurrency should work: it needs to be possible to trust no one on the network and yet have the network still work.

A non-trustless design like this should be a non-starter.

---

## Discussion Thread

### Comment by @softarch24

**Date:** 2019-01-11

Regarding "Privacy leakage" and "This design is non-trustless" comments -
Yes, the proxies have some insight on details of payments (note - we are talking about merchant payments, not regular P2P transfers). The idea behind proxy is that it takes care of some operations that are difficult or impossible to implement on mobile device, especially with tough requirements of CryptoNote protocol. The proxy is somewhat trusted; however, it can be either public (as a service provided by trusted third party service provider to multiple merchants) or proprietary (as a local supernode that belongs to the single merchant). For most merchants, it is more important to get best levels of service than absolute privacy. In case absolute secrecy is required, the merchant can run its proprietary proxy.  

---

### Comment by @softarch24

**Date:** 2019-01-11

Regarding "selfish mining double-spending attack" -
This is known attack on PoW blockchains called "Finney attack": https://bitcoin.stackexchange.com/questions/4942/what-is-a-finney-attack
GRAFT is not the only PoW blockchain that is vulnerable to this attack. 
For RTA, we are going to implement locking mechanism similar to the one implemented by DASH. Once RTA Tx is authorized by the authorization sample, the Tx is broadcasted to the entire network. If an attacker injects a block (or chain) containing Tx that conflicts with the locked Tx (i.e. trying to spend the same key images), such a block (or chain) will be rejected (see section 4.2 Finney Attacks):  
https://github.com/dashpay/docs/blob/master/binary/Dash%20Whitepaper%20-%20Transaction%20Locking%20and%20Masternode%20Consensus.pdf (see 
In addition, DASH has recently suggested another protection mechanism that mitigates 51% mining attack even on regular (non-instant) Tx, which essentially makes even a regular transfer transaction irreversible after 1 confirmation:
https://github.com/dashpay/dips/blob/master/dip-0008.md
We are weighing our options of implementing a similar mechanism in the future.  


---

### Comment by @jagerman

**Date:** 2019-01-12

> Yes, the proxies have some insight on details of payments (note - we are talking about merchant payments, not regular P2P transfers).

It is unnecessary and undermines the privacy that less than two months ago [you posted about](https://www.graft.network/2018/11/21/how-graft-is-similar-to-and-at-the-same-time-different-from-visa-and-other-payment-card-networks-part-2/) as being a key difference in the GRAFT payment network:

> ###  Difference #2 – Privacy
> Another key difference is ... absolute privacy provided by GRAFT Network to both buyer and merchant. Unlike plastic cards and most cryptocurrencies, GRAFT’s sender address, recipient address, transaction amount, and transaction fee amount are invisible to everyone except for the sender and recipient themselves. Although payment card networks do not expose the details of transaction to the public, this data is accessible by employees of multiple corporations, can be shared with governments, and can be stolen by hackers.

But now you are saying:

> For most merchants, it is more important to get best levels of service than absolute privacy.

And that merchants who actually want the proclaimed privacy will have to have the expertise to run, update and keep secure their own proxy SN.

> The idea behind proxy is that it takes care of some operations that are difficult or impossible to implement on mobile device, especially with tough requirements of CryptoNote protocol.

What operations, exactly, do you think cannot be done on mobile hardware?  Are you not aware of mobile wallets for several cryptonote coins such as [monerujo (for Monero)](https://play.google.com/store/apps/details?id=com.m2049r.xmrwallet&hl=en), [Loki Wallet](https://play.google.com/store/apps/details?id=network.loki.wallet&hl=en_US), or [Haven Protocol Wallet](https://itunes.apple.com/us/app/haven-protocol-wallet/id1438566523?ls=1&mt=8), to name just a few, which are able to handle CryptoNote just fine without leaking privacy and security to a remote proxy?  Or that a Raspberry Pi (which has essentially the same computational power as the slowest Verifone Carbon device) is perfectly capable of running not only dozens of CryptoNote wallets simultaneously, but also multiple whole cryptonode nodes simultaneously?

> The proxy is somewhat trusted

No, it is not "somewhat" trust.  It is entirely trusted.  In this design, the proxy SN is the one that tells the merchant *without verifiable proof* that a payment has been approved by the network.  It is a huge target for attacks and said attacks will be difficult to detect until long after the fact.  This single point of attack effectively undermines the entire security of the RTA mechanism, to the point where you might as well not even *have* RTA: you could literally do the entire authorization in just the proxy SN and have just as much security as you are getting here because your weakest link would be the same.

The entire point of using a random sample on a decentralized network is the security it brings, because someone would have to own or compromise a very large share of the network in order to compromise the security of the network.  Hacking an RTA supernode or coercing its operator would gain you absolutely nothing.  The design in this RFC, however, specifies a trusted, centralized component that must exist in every single RTA transaction; a component that can be hacked or have its operator coerced to compromise the security and privacy of any and all merchants using that node.

This is not an responsible or acceptable design.

---

### Comment by @SomethingGettingWrong

**Date:** 2019-01-12

**RTA OF ANY PRIVACY CRYPTO SHOULD BE PRIVATE**

The privacy of any crypto is the number one community backed assumption and choice that a project should take the steps to complete when they support it! Otherwise you should have just forked Dash! which was based off of bitcoin.

Just because It technically works at RTA doesn't mean you will have the support of the community. If the community doesn't support it then the price will dump to the cost of mining it! which will further go down as difficulty lowers as miners leave as the price drops!

*What you are trying to achieve could have been achieved; while , at the same time staying private.*

I fear that you thought privacy had to be sacrificed in order to make it compatible with merchants terminals. When indeed that is not the case! I feel this came about from a lack of understanding the actual fundamental privacy of the Monero blockchain and from not listening to the community who was practicly screaming! Please Please Please don't implement it this way!

Now you have "completed" an Alpha that while technicly does RTA yet it has no privacy and is insecure with a central failure point the proxy supernode. Which by definition means its not decentralized

**You guys are busy implementing all these new features working on them all at one time! Instead of just sticking to something the community would have wanted and what we thought it was!**

**A Privacy/RTA coin.**

You guys are programming this as if no one will modify super node code for nefarious purposes! All the risk is left on the super nodes running this code! While we would be okay with that if it was all anonymous/secure. The fact of the matter is your leaving it unprivate and and insecure and leaving the burden of running the code on the users and their stake amount while telling everyone its private!

maybe if you would have not been so in the dark about it's development and decisions and had more community involvement the project would corrected itself!

**You had plenty of opensource developers who would have helped you if you would have just listend and done it a different way. Instead you thought it could only be done this way. when we are telling you if you do it this way your making a mistake**

You are running it as if its closed source software! That mentality has caused you to sacrifice the security and privacy when programming. Instead of actually listening to the community you pushed your community developers away. Just because you know how to program and you understand Merchant terminals doesn't mean you comprehend privacy blockchain! If you do and you implemented this anyway "SHAME ON YOU"

_All your answers are we are right you are wrong and this is why! or you say.. I don't see the issue can we close this?_

Reading this code has me baffled! Its not even the programmers. I feel its the way the team is telling them to implement it and I feel the team doesn't realize this is a mistake and are in denial because they have spent so much time going this direction!

Its not too late to turn around yah know! The direction you are taking this is away from the community.. which means no one will use it! Have you not noticed community is dissolving?


---

### Comment by @necro-nemesis

**Date:** 2019-01-13

RTA must have end to end encryption for the protection of node owners. Zero knowledge proof of knowledge. Disclosing information to a node presents unlimited liability for whomever operates it. Anyone who understands this will not operate a node since the risks greatly outweigh the benefits.

---

### Comment by @SomethingGettingWrong

**Date:** 2019-01-17

@sgomzin  

Please create your own unique algo or "tweak" another algo that's lesser known like XTL or Haven.  
(more gpu's can support xtl variant) but at this point a v8 tweak would be fastest


**STOP WEIGHING YOUR OPTIONS AND PICK ONE!**

**[P2P6]  INFO   global  src/cryptonote_core/blockchain.cpp:933  REORGANIZE SUCCESS! on height: 263338, new blockchain size: 263442**

Any top exchange would delist! It would not surprise me if Cryptopia and Tradeogre 
delists you guys. 

You need to reevaluate  your understanding of a 51 percent attack!   

I warned him.. we will see how it goes. (not looking good)

The blockchain should have a checkpoint every few blocks or something when below such a hashrate. I cant think of any situation where you would need to reorganize more then 20 blocks. 

![image](https://user-images.githubusercontent.com/36722911/51296184-75b9f280-19e0-11e9-9ce9-7741896a567c.png)


---

### Comment by @bitkis

**Date:** 2019-01-19

@jagerman Thanks for the valuable and constructive criticism.

> So, to summarize the privacy leaks that seem to be here:
>
> * the PoS proxy SN sees the recipient wallet address, the total amount, and individual items purchased including the amount of each item.
> * auth sample SNs see the total amount including the amount received by the proxy PoS
> * wallet proxy SN plus, apparently, any SN can get an itemized list of the transaction

The RFC is updated, we tried to address most of the concerns. Note that though the total amount is still open, no association between transaction and recipient wallet address can be built.

> this design has no protection against a selfish mining double-spending attack. Unlike a double-spending attack against an exchange, double-spending here does not have to reach any minimum number of confirmations; and can be timed (with a little effort) to not even require 51% of the network. (I pointed this out just over two months ago in the public JIRA with details of how to carry out an attack and a demo but the issue has had no response).

We know it's an open issue and still weighing our options here.

> > 12. Once the graftnode accepts the transaction, supernode, which submitted it to the cryptonode, broadcasts successful pay status over the network

> Why is this needed at all? Success can already been seen (and is already transmitted across the network) by the fact that the transaction enters the mempool. Can't the wallet just check for that instead?

It's a work around the fact we could often observe mempool sync required extra time.

---

### Comment by @SomethingGettingWrong

**Date:** 2019-01-21

@bitkis What options are you weighing? Super node consensus seems to be the way dash and Loki are handling similar things. I would do something similar. 

---

