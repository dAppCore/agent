# PR #225: Blockchain based list implementation

## Reception Score

| Score | Reason |
|-------|--------|
| **MERGED** | Contribution accepted |

---

## Metadata

| Field | Value |
|-------|-------|
| State | MERGED |
| Author | @LenyKholodov |
| Created | 2019-02-04 |
| Merged | 2019-03-05 |

---

## Description

Blockchain based list is used for building list of supernodes which may be used for further authentication.

Implementation details:
* list is built for every block based on it's hash and active stake transactions;
* block hash is used as a bye array for selecting supernodes from active supernodes (in terms of stake validity time);
* the list is stored to file after each update;
* the list is loaded during cryptonode start from a file (if it exists).

---

## Reviews & Comments

### Comment by @jagerman

The sample selection being done here to select a blockchain-based supernode tier subset is non-uniform, and results in relatively small samples.  It is also entirely non-obvious why these lists are being reduced to a random subset in the first place.

To deal with the latter issue first: with a hard cap on the number of supernodes selected into a sample you are effectively limiting the scalability of the network.  More supernodes active at a time will add no additional capability to the network because at each block you cut down the list of supernodes that are available to handle SN operations.  Why is this being done?  If you were to pass the entire list of active supernodes on each tier to the supernode and let it randomly sample from that list (based on the payment ID) it would be far more scalable.

Now as for the former issue.  Since the source vector from which elements are sampled is itself sorted by the age of the stake, this whole process results in non-uniform selection: some supernodes have a greater chance of selection than others (and depending on the counts, some have no probability of being selected at all).  For example, when you have 50 supernodes on a tier you get `PREVIOS_BLOCKCHAIN_BASED_LIST_MAX_SIZE` selected from the previous block list (why?), plus another 32 selected from using the randomization algorithm (since you are using the `char` of the block hash as your RNG, and only have 32 `char`s to work with).  When I use your algorithm to look at the frequency of selection of the 50 nodes, I get this:

```
Selection frequency: (uniform frequency: 0.64):
[  0]: 0.715325
[  1]: 0.714514
[  2]: 0.719117
[  3]: 0.723792
[  4]: 0.727855
[  5]: 0.731591
[  6]: 0.734153
[  7]: 0.73704
[  8]: 0.738946
[  9]: 0.741059
[ 10]: 0.742394
[ 11]: 0.743742
[ 12]: 0.744824
[ 13]: 0.745515
[ 14]: 0.746299
[ 15]: 0.746988
[ 16]: 0.690373
[ 17]: 0.671085
[ 18]: 0.658806
[ 19]: 0.65022
[ 20]: 0.643962
[ 21]: 0.639378
[ 22]: 0.635563
[ 23]: 0.633008
[ 24]: 0.630666
[ 25]: 0.629243
[ 26]: 0.628241
[ 27]: 0.627435
[ 28]: 0.57412
[ 29]: 0.547461
[ 30]: 0.531217
[ 31]: 0.520952
[ 32]: 0.513832
[ 33]: 0.509343
[ 34]: 0.506473
[ 35]: 0.504151
[ 36]: 0.502728
[ 37]: 0.501716
[ 38]: 0.561549
[ 39]: 0.584621
[ 40]: 0.59685
[ 41]: 0.604984
[ 42]: 0.610537
[ 43]: 0.614386
[ 44]: 0.61711
[ 45]: 0.618959
[ 46]: 0.62066
[ 47]: 0.621801
[ 48]: 0.622307
[ 49]: 0.623108
```
(These values are based on 10M repetitions of the algorithm, where each `extract_index` uses a value drawn from `static std::uniform_int_distribution<char> random_char{std::numeric_limits<char>::min(), std::numeric_limits<char>::max()};`.  Typical variation across runs here is in the 4th decimal place: this is not a sampling aberration.)

This is very clearly not a uniform distribution: the 15th-oldest supernode has almost 50% higher probability of being selected compared to the 38th oldest.

For other supernode numbers things get worse; here's the sampling frequency when there are 250 supernodes on a tier:

```
[  0]: 0.24291
[  1]: 0.24728
[  2]: 0.249168
[  3]: 0.249518
[  4]: 0.249791
[  5]: 0.250054
[  6]: 0.250062
[  7]: 0.24979
[  8]: 0.249791
[  9]: 0.249997
[ 10]: 0.249981
[ 11]: 0.249963
[ 12]: 0.250104
[ 13]: 0.249791
[ 14]: 0.250034
[ 15]: 0.250051
[ 16]: 0.250057
[ 17]: 0.250055
[ 18]: 0.249884
[ 19]: 0.25012
[ 20]: 0.250039
[ 21]: 0.250088
[ 22]: 0.250208
[ 23]: 0.250117
[ 24]: 0.250177
[ 25]: 0.249837
[ 26]: 0.249773
[ 27]: 0.249865
[ 28]: 0.250205
[ 29]: 0.250166
[ 30]: 0.250068
[ 31]: 0.249756
[ 32]: 0.249978
[ 33]: 0.24987
[ 34]: 0.250209
[ 35]: 0.249829
[ 36]: 0.250101
[ 37]: 0.250132
[ 38]: 0.250032
[ 39]: 0.24971
[ 40]: 0.249928
[ 41]: 0.249834
[ 42]: 0.250064
[ 43]: 0.250113
[ 44]: 0.250229
[ 45]: 0.249869
[ 46]: 0.249862
[ 47]: 0.250021
[ 48]: 0.249953
[ 49]: 0.250074
[ 50]: 0.250051
[ 51]: 0.249851
[ 52]: 0.249894
[ 53]: 0.249789
[ 54]: 0.24987
[ 55]: 0.250084
[ 56]: 0.249922
[ 57]: 0.250097
[ 58]: 0.250028
[ 59]: 0.250173
[ 60]: 0.249823
[ 61]: 0.250085
[ 62]: 0.249914
[ 63]: 0.25002
[ 64]: 0.250072
[ 65]: 0.24988
[ 66]: 0.250086
[ 67]: 0.250092
[ 68]: 0.249764
[ 69]: 0.249885
[ 70]: 0.250143
[ 71]: 0.249959
[ 72]: 0.249907
[ 73]: 0.249892
[ 74]: 0.249984
[ 75]: 0.249953
[ 76]: 0.250395
[ 77]: 0.250094
[ 78]: 0.250099
[ 79]: 0.249982
[ 80]: 0.250033
[ 81]: 0.249815
[ 82]: 0.249907
[ 83]: 0.250006
[ 84]: 0.249939
[ 85]: 0.249977
[ 86]: 0.250034
[ 87]: 0.250029
[ 88]: 0.249932
[ 89]: 0.250139
[ 90]: 0.250167
[ 91]: 0.250096
[ 92]: 0.249912
[ 93]: 0.250008
[ 94]: 0.250053
[ 95]: 0.249949
[ 96]: 0.250287
[ 97]: 0.250034
[ 98]: 0.249838
[ 99]: 0.250176
[100]: 0.250165
[101]: 0.250049
[102]: 0.249944
[103]: 0.250206
[104]: 0.25
[105]: 0.250052
[106]: 0.250005
[107]: 0.250039
[108]: 0.249936
[109]: 0.250015
[110]: 0.249985
[111]: 0.249776
[112]: 0.249764
[113]: 0.250092
[114]: 0.249951
[115]: 0.24985
[116]: 0.134431
[117]: 0.126543
[118]: 0.1252
[119]: 0.125071
[120]: 0.125212
[121]: 0.124933
[122]: 0.124989
[123]: 0.124869
[124]: 0.125012
[125]: 0.125022
[126]: 0.124945
[127]: 0.124973
[128]: 0.0081291
[129]: 0.0003719
[130]: 1.37e-05
[131]: 6e-07
[132]: 0
[133]: 0
[134]: 0
[135]: 0
[136]: 0
[137]: 0
[138]: 0
[139]: 0
[140]: 0
[141]: 0
[142]: 0
[143]: 0
[144]: 0
[145]: 0
[146]: 0
[147]: 0
[148]: 0
[149]: 0
[150]: 0
[151]: 0
[152]: 0
[153]: 0
[154]: 0
[155]: 0
[156]: 0
[157]: 0
[158]: 0
[159]: 0
[160]: 0
[161]: 0
[162]: 0
[163]: 0
[164]: 0
[165]: 0
[166]: 0
[167]: 0
[168]: 0
[169]: 0
[170]: 0
[171]: 0
[172]: 0
[173]: 0
[174]: 0
[175]: 0
[176]: 0
[177]: 0
[178]: 0
[179]: 0
[180]: 0
[181]: 0
[182]: 0
[183]: 0
[184]: 0
[185]: 0
[186]: 0
[187]: 0
[188]: 0
[189]: 0
[190]: 0
[191]: 0
[192]: 0
[193]: 0
[194]: 0
[195]: 0
[196]: 0
[197]: 0
[198]: 0
[199]: 0
[200]: 0
[201]: 0
[202]: 0
[203]: 0
[204]: 0
[205]: 0
[206]: 0
[207]: 0
[208]: 0
[209]: 0
[210]: 0
[211]: 0
[212]: 0
[213]: 0
[214]: 0
[215]: 0
[216]: 0
[217]: 0
[218]: 0
[219]: 0
[220]: 0
[221]: 0
[222]: 0
[223]: 0
[224]: 0
[225]: 0
[226]: 0
[227]: 0
[228]: 0
[229]: 0
[230]: 0
[231]: 0
[232]: 0
[233]: 0
[234]: 0
[235]: 0
[236]: 0
[237]: 0
[238]: 0.117817
[239]: 0.124049
[240]: 0.124957
[241]: 0.125015
[242]: 0.125061
[243]: 0.124996
[244]: 0.125086
[245]: 0.125103
[246]: 0.124908
[247]: 0.124911
[248]: 0.125068
[249]: 0.124864
```

Another strange thing happening in this algorithm is that it never selects more than 32 supernodes for a tier (because there are only 32 `char`s in the block hash), but once there are 256 or more supernodes, you start selecting only 16 per block.  (These get added to `PREVIOS_BLOCKCHAIN_BASED_LIST_MAX_SIZE` selected from the previous sample, so technically it is going to build a list of 33 SNs for a tier with up to 255 SNs on it, and 17 SNs for a tier with >= 256).

The `PREVIOS_BLOCKCHAIN_BASED_LIST_MAX_SIZE` also makes no sense here: what is gained by keeping a subset of the previous round's subset in the list of available SNs?


# Why?

I am left asking: why are you doing all of this?

This approach (combined with https://github.com/graft-project/graft-ng/pull/204) results in a non-uniform, hard-capped number of SNs to select from each tier.

You can make a simpler, far more robust, _uniform_ sampling algorithm by just giving the SN *all* of the supernodes on each tier, then using the payment ID to seed a PRNG (like `std::mt19937_64`) and using this to randomly sample from each tier.

That's not ideal, though, because it can be gamed: I could use a supernode to reroll payment IDs until I get one that favours my own SNs.  You can work around that fairly easily doing something like this:

1. Don't do any sampling in GraftNetwork; instead just provide the entire list of supernodes currently active at each tier along with the relevant block hash value.
2. Inside graft-ng, generate a payment-id.
3. Hash the payment-id together with the block hash.
4. Use that resulting hashed value to seed a `std::mt19937_64`.
5. Use this RNG to sample 2 supernodes from each tier.

The harder you make step 3 the more costly it is to game the system (but also, the more costly it becomes to verify).  The block hash from step 1 is needed in step 2 so that you can't pregenerate lots of payment IDs offline with known SN selection positions in advance.

And all of this is *still* going to be significantly less code than you are using now to generate a badly broken sample.

---

### Comment by @LenyKholodov

Jason, thank you for your feedback. We will check the results you kindly provided and return to you soon.

---

### Comment by @LenyKholodov

> Jason, thank you for your feedback. We will check the results you kindly provided and return to you soon.

@jagerman Could you please repeat your test with following fix?

```
size_t extract_index(const char* it, size_t length)
{
  size_t result = 0;

  for (;length--; it++)
    result = (result << 8) + size_t(*reinterpret_cast<const unsigned char*>(it));

  return result;
}
```

---

### Comment by @jagerman

Changing it from a signed to unsigned char gets rid of the hole above 128, but doesn't fix the non-uniformity of the distribution; for 200 nodes it now results in the first few having these probabilities:

```
[  0]: 0.228301
[  1]: 0.243768
[  2]: 0.248024
[  3]: 0.249059
[  4]: 0.249682
[  5]: 0.250019
[  6]: 0.149295
[  7]: 0.130186
[  8]: 0.126137
[  9]: 0.125245
[ 10]: 0.12497
```
with the remaining 11-249 all being close to 0.125.

---

### Comment by @jagerman

The unsigned results for N=50 show the same pattern: too high selection probability on the first 10-15 elements and slightly too low on the remaining ones.

The reason is pretty simple: `random_value % N` does *not* produce a uniform distribution over [0, *N*-1], though it does get close if *N* is much larger than `random_value` by at least a couple orders of magnitude.

If you absolutely need to construct a deterministic random selection here (but I really don't think you do or *should*--see my comments above) you are best off generating values from a single `std::mt19937_64` that you seed using a `std::uint_fast64_t` value constructed from the hash.

You also need to drop the `offset` addition from `(offset + random_value) % src_list_size`--this is biasing the selection probability away from the first elements (which is why in the above example you see an increase in probabilities over the first few elements).

Actually, on that note, if you absolutely must keep random sampling here (and again, I don't see any reason why you would need this!) I think you should scrap the whole thing and use this far more algorithmically efficient approach to select m of n values with linear (O(n)) complexity (your current implementation looks to me to be O(mn²)): https://stackoverflow.com/questions/136474/best-way-to-pick-a-random-subset-from-a-collection/136730#136730

---

### Comment by @LenyKholodov

@jagerman We have prepared two tests with implementation of blockchain based list which can be run separately.
- our current implementation - https://github.com/graft-project/GraftNetwork/blob/blockchain_based_list_tests/test_blockchain_based_list.cpp - it has behavior which you have described above (first 10 nodes are elected more often than others); 
- Mersenne Twister implementation - https://github.com/graft-project/GraftNetwork/blob/blockchain_based_list_tests/test_mersenne_twister.cpp - fully random, but much slower.

Mersenne Twister provides really uniform distribution but has worse performance compared to blockchain based list building implementation based on block hash indexes.

We don't set the goal to achieve theoretically uniform distribution so for balancing it's fully ok to have first 10 nodes with higher probabilities than other 200+ during selection of nodes to a blockchain based list. Also, in the test we use static list of supernodes for selection (as we understood you did the same). In a real environment for 10M blocks it will be impossible to have static list of supernodes for selection, first of all because we are limiting stake transaction lock time. So we expect randomness will be achieved by stake transaction generation and by block hashes (then also by payment IDs during auth sample building). Also, we are making simulation on top of current blockchain based implementation with real block hashes to find out values of parameters. So their current values are not final.

In one of your previous comments you were absolutely correct that it's no acceptable to have supernodes with zero probability to be selected in a blockchain based list. This was implementation bug which was related to incorrect conversion from signed char to unsigned int.

We are discussing usage of Mersenne Twister implementation instead of current implementation. However, at this time we don't see advantages why it should be used instead of current model.

---

### Comment by @jagerman

First point: I never suggested using `std::uniform_int_distribution`, and in fact you should *not* use it here because it doesn't have C++-standard-guaranteed results.  (It also slows things down slightly).

Second point:
> We don't set the goal to achieve theoretically uniform distribution so for balancing it's fully ok to have first 10 nodes with higher probabilities than other 200+ during selection of nodes to a blockchain based list.

is just plain wrong: it is not okay.  From the whitepaper:

> Each tier participates in a random selection of 2 sample supernodes.

While a non-uniform sample that probabilistically provides higher rewards to supernodes within a tier that were registered earlier to ones registered later is still, in a technical sense, "random", it is most definitely *not* what most people would assume the whitepaper means by "random."


Third, if your code is running slowly, it's highly unlikely that `std::mt19937_64` (nor `std::mt19937` which you used instead) is the cause:

### r.cpp
```C++
#include <random>
#include <cstdint>
#include <iostream>
#include <chrono>

constexpr size_t ITERS = 100000000;
int main() {
    std::mt19937_64 rng;
    std::uint64_t x = 0;
    auto start = std::chrono::high_resolution_clock::now();

    std::uint64_t count = 250;

    for (size_t i = 0; i < ITERS; i++)
        x += rng() % count;

    auto end = std::chrono::high_resolution_clock::now();
    auto elapsed_us = std::chrono::duration_cast<std::chrono::microseconds>(end - start).count();
    uint64_t dps = static_cast<uint64_t>(double(ITERS) / elapsed_us * 1000000);
    std::cout << ITERS << " values drawn in " << elapsed_us << "µs = " << dps << " draws per second\n";
    std::cout << "\n(meaningless sum of all draws = " << x << ")\n";
}
```

Results:
```
betwixt:~$ g++ -O2 r.cpp -o r
betwixt:~$ ./r
100000000 values drawn in 640173µs = 156207775 draws per second

(meaningless sum of all draws = 12450205566)
```

`std::mt19937_64` is not a performance limitation here.

---

### Comment by @jagerman

> We are discussing usage of Mersenne Twister implementation instead of current implementation. However, at this time we don't see advantages why it should be used instead of current model.

I actually (sort of) agree with this.  You should not have any sampling *at all* in graftnoded.  The entire sampling process can be done *once* in graft-ng incorporating both the entropy in the current block hash *and* the entropy in the payment id.

---

### Comment by @yidakee

@LenyKholodov -  if you don't mind me saying so, please be mindful of wording. 

"We don't set the goal to achieve theoretically uniform distribution so for balancing it's fully ok to have first 10 nodes with higher probabilities than other 200+ during selection of nodes to a blockchain based list."

This is the furthest from a fair and evenly distributed network. If think (I hope) what you means is that, currently, an even distribution is not on the top of the list on the development backlog (why not?) but that balancing is what is currently being worked on, and after that we will implement a fair distribution model.

This is 100% of the objective - to achieve an equalitarian Supernode distribution. Otherwise the system can and will be gamed, and adoption will not follow.

---

### Comment by @jagerman

> - our current implementation - https://github.com/graft-project/GraftNetwork/blob/blockchain_based_list_tests/test_blockchain_based_list.cpp - it has behavior which you have described above (first 10 nodes are elected more often than others);
> - Mersenne Twister implementation - https://github.com/graft-project/GraftNetwork/blob/blockchain_based_list_tests/test_mersenne_twister.cpp - fully random, but much slower.

Your "current implementation" selects 32 supernodes out of 250 while you make the Mersenne twister implementation select 255 out of 255 (and in doing so you end up hitting the worse case performance of your implementation algorithm).  The result is even apparent in your output: every index is selected with probability of exactly 1.

Here's a proper implementation that fairly compares: https://jagerman.com/test_mersenne_twister.cpp by selecting 32/250 (I also increased the number of experiments back to 100k):

```
Results after 100000 experiments: 
 f[000]: 12748 0.127480
 f[001]: 12852 0.128520
... (many more all 0.127xxx or 0.128xxx -- theoretical ideal is 0.1280000)
 f[249]: 12812 0.128120

real	0m0.708s
user	0m0.707s
sys	0m0.000s
```

Here's yours:
```
Results after 100000 experiments:
 f[000]: 0.227360
 f[001]: 0.246580
 f[002]: 0.249790
 f[003]: 0.248780
 f[004]: 0.248810
 f[005]: 0.248990
 f[006]: 0.147330
 f[007]: 0.130810
 f[008]: 0.126130
 f[009]: 0.126050
 f[010]: 0.125840
 f[011]: 0.125440
... (various values between 0.123xxx and 0.126xxx; theoretical ideal is 0.128000)
 f[249]: 0.124110

real	0m0.276s
user	0m0.275s
sys	0m0.000s
```

---

### Comment by @LenyKholodov

> @LenyKholodov - if you don't mind me saying so, please be mindful of wording.
> 
> "We don't set the goal to achieve theoretically uniform distribution so for balancing it's fully ok to have first 10 nodes with higher probabilities than other 200+ during selection of nodes to a blockchain based list."
> 
> This is the furthest from a fair and evenly distributed network. If think (I hope) what you means is that, currently, an even distribution is not on the top of the list on the development backlog (why not?) but that balancing is what is currently being worked on, and after that we will implement a fair distribution model.
> 
> This is 100% of the objective - to achieve an equalitarian Supernode distribution. Otherwise the system can and will be gamed, and adoption will not follow.

@yidakee Thank you for your feedback. All tests which are discussing in this thread have assumption that the list supernodes with stake is static during the whole test of thousands of iteration. In practice blockchain based list is built for each block so for example 10k iterations is equal to 10k blocks and it is  impossible to have fully static stake supernodes list during 10k blocks. That's why we don't see big issue with non equal probabilities of supernodes for blockchain based list. This is only one of three existing random layers:
1) generation of stakes and list of supernodes with stakes;
2) blockchain based list based on the result of step (1) which is discussed in this PR;
3) auth sample generation based on result of step (2).

---

### Comment by @LenyKholodov

> First point: I never suggested using `std::uniform_int_distribution`, and in fact you should _not_ use it here because it doesn't have C++-standard-guaranteed results. (It also slows things down slightly).

I didn't write that you suggested uniform_int_distribution. However, for the test it is not so important. Any other uniform distribution generator may be used to check probabilities of generated supernodes indexes. So uniform_int_distribution is only a tool.

> 
> Second point:
> 
> > We don't set the goal to achieve theoretically uniform distribution so for balancing it's fully ok to have first 10 nodes with higher probabilities than other 200+ during selection of nodes to a blockchain based list.
> 
> is just plain wrong: it is not okay. From the whitepaper:
> 
> > Each tier participates in a random selection of 2 sample supernodes.
> 
> While a non-uniform sample that probabilistically provides higher rewards to supernodes within a tier that were registered earlier to ones registered later is still, in a technical sense, "random", it is most definitely _not_ what most people would assume the whitepaper means by "random."

Please keep in mind that we use three layers of randomness:
1) stakes generation;
2) blockchain based list with block hash as a random value;
3) auth sample building with payment ID as a random value.
Also, current implementation provides only a model without configured parameters. We are testing it now and will update with parameters which lead of uniform distribution of auth sample.
 
> 
> Third, if your code is running slowly, it's highly unlikely that `std::mt19937_64` (nor `std::mt19937` which you used instead) is the cause:
> 
> ### r.cpp
> ```c++
> #include <random>
> #include <cstdint>
> #include <iostream>
> #include <chrono>
> 
> constexpr size_t ITERS = 100000000;
> int main() {
>     std::mt19937_64 rng;
>     std::uint64_t x = 0;
>     auto start = std::chrono::high_resolution_clock::now();
> 
>     std::uint64_t count = 250;
> 
>     for (size_t i = 0; i < ITERS; i++)
>         x += rng() % count;
> 
>     auto end = std::chrono::high_resolution_clock::now();
>     auto elapsed_us = std::chrono::duration_cast<std::chrono::microseconds>(end - start).count();
>     uint64_t dps = static_cast<uint64_t>(double(ITERS) / elapsed_us * 1000000);
>     std::cout << ITERS << " values drawn in " << elapsed_us << "µs = " << dps << " draws per second\n";
>     std::cout << "\n(meaningless sum of all draws = " << x << ")\n";
> }
> ```
> 
> Results:
> 
> ```
> betwixt:~$ g++ -O2 r.cpp -o r
> betwixt:~$ ./r
> 100000000 values drawn in 640173µs = 156207775 draws per second
> 
> (meaningless sum of all draws = 12450205566)
> ```
> `std::mt19937_64` is not a performance limitation here.

Thank you very much for these results. We will check them.


---

### Comment by @LenyKholodov

We checked current blockchain based list implemented and found that it may also be easily modified to achieve uniform distribution requirement. Please find updated source here - https://github.com/graft-project/GraftNetwork/blob/98ab487fdb7482ff6d3792e6c9df6bf0a290ddb5/test_blockchain_based_list.cpp

---

### Comment by @jagerman

> 1. stakes generation

This is not random since people can act to influence it.

> 3. auth sample building with payment ID as a random value.

It is completely irrelevant whether this stage is random or not because the step we are discussing *here* throws away elements from consideration in that stage with non-uniform probability.  The fact that you later on randomize among the elements that don't get thrown away does *nothing* to change that: they don't make it to this stage at all.  (They should, but you seem to prefer to simply ignore that point).

> We checked current blockchain based list implemented and found that it may also be easily modified to achieve uniform distribution requirement. Please find updated source here - https://github.com/graft-project/GraftNetwork/blob/98ab487fdb7482ff6d3792e6c9df6bf0a290ddb5/test_blockchain_based_list.cpp

It is better, though there is still a significant problem with it that I mentioned earlier: it is not capable of selecting more than 32 supernodes, and worse, once the network hits 257 supernodes on a tier it actually has to *reduce* the work size sample from 32 to 16 supernodes per tier.  You can probably fix it, but what's the point when you have a superior solution with known statistical properties right in front of you that *simplifies* your code?

I do not understand your resistance here: `std::mt19937_64` (or even `std::minstd_rand` if you prefer) are well understood algorithms with good performance (a bit better for `std::minstd_rand`), excellent statistic properties (much better for `std::mt19937_64`), are included in the C++ standard, are entirely deterministic for any given seed, do not impose a significant performance cost, result in simpler code, and do not impose any restriction on the number of supernodes that can be selected.

You've thrown up obstacles, you've ignored half of what I've said (most notably why you want randomness at this stage *at all*), and you produced a faulty benchmark to try to prove a technical deficit that doesn't exist.

Please start considering this issue on *technical* grounds rather than emotional ones.

---

### Comment by @LenyKholodov

> > 1. stakes generation
> 
> This is not random since people can act to influence it.
> 
> > 1. auth sample building with payment ID as a random value.
> 
> It is completely irrelevant whether this stage is random or not because the step we are discussing _here_ throws away elements from consideration in that stage with non-uniform probability. The fact that you later on randomize among the elements that don't get thrown away does _nothing_ to change that: they don't make it to this stage at all. (They should, but you seem to prefer to simply ignore that point).
> 
> > We checked current blockchain based list implemented and found that it may also be easily modified to achieve uniform distribution requirement. Please find updated source here - https://github.com/graft-project/GraftNetwork/blob/98ab487fdb7482ff6d3792e6c9df6bf0a290ddb5/test_blockchain_based_list.cpp
> 
> It is better, though there is still a significant problem with it that I mentioned earlier: it is not capable of selecting more than 32 supernodes, and worse, once the network hits 257 supernodes on a tier it actually has to _reduce_ the work size sample from 32 to 16 supernodes per tier. You can probably fix it, but what's the point when you have a superior solution with known statistical properties right in front of you that _simplifies_ your code?
> 
> I do not understand your resistance here: `std::mt19937_64` (or even `std::minstd_rand` if you prefer) are well understood algorithms with good performance (a bit better for `std::minstd_rand`), excellent statistic properties (much better for `std::mt19937_64`), are included in the C++ standard, are entirely deterministic for any given seed, do not impose a significant performance cost, result in simpler code, and do not impose any restriction on the number of supernodes that can be selected.
> 
> You've thrown up obstacles, you've ignored half of what I've said (most notably why you want randomness at this stage _at all_), and you produced a faulty benchmark to try to prove a technical deficit that doesn't exist.
> 
> Please start considering this issue on _technical_ grounds rather than emotional ones.

@jagerman Thank you very much for your detailed feedback.

> Please start considering this issue on _technical_ grounds rather than emotional ones.

I believe I've been discussing technical issues through the whole discussion without any emotions. If you see any emotions from my side, please forgive me. Emotions is not that I usually use. Current implementation is based on technical vision (https://github.com/graft-project/graft-ng/wiki/%5BRFC-002-SLS%5D-Supernode-List-Selection). We are grateful to you for your vision and proposal and still discussing it internally, but at this time we don't see any advantages of using one of pseudo random implementations. Both algorithms MT and current supernodes selection use same source of entropy - block hash. As you correctly noted original PR had technical issues which led to non uniform distribution of supernodes selection. We are fixing them now.

> You've thrown up obstacles, you've ignored half of what I've said (most notably why you want randomness at this stage _at all_), and you produced a faulty benchmark to try to prove a technical deficit that doesn't exist.

I'm not ignoring what you wrote here. However, at this time the main issue which we're focusing is distribution of blockchain based building. That's why some questions may remain unanswered now.

> why you want randomness at this stage _at all_

We expect to have thousands of valid stake transactions and as a result thousands of active supernodes. We need to select small subset of supernodes which will be potentially used for auth samples during one block. There will be rules about connection management of supernodes in the subset which are not yet described in public. However, the main thing here is that we want to select and fix small subset of supernodes (16-30) for the block. Then this subset will be used as a source for selecting auth sample during the payments based on RTA payment ID as a random source. So for each payment only several nodes from the subset will be used.

>  It is better, though there is still a significant problem with it that I mentioned earlier: it is not capable of selecting more than 32 supernodes, and worse, once the network hits 257 supernodes on a tier it actually has to _reduce_ the work size sample from 32 to 16 supernodes per tier. 

We don't expect to have more than 32 nodes in a blockchain based list. However, there is no problem to increase it if needed. One of the simplest solution is to use previous block hashes in some combination with current block hash.

> I do not understand your resistance here: `std::mt19937_64` (or even `std::minstd_rand` if you prefer) are well understood algorithms with good performance (a bit better for `std::minstd_rand`), excellent statistic properties (much better for `std::mt19937_64`), are included in the C++ standard, are entirely deterministic for any given seed, do not impose a significant performance cost, result in simpler code, and do not impose any restriction on the number of supernodes that can be selected.

It's very simple. At this time we are implementing and testing solution which is based on previously described technical vision (which I mentioned above in this comment). From our point of view, comparison of random generators may be used only in terms of simplicity and distribution. There are many others well known RNG implementation. However, as I wrote earlier we don't see significant advantages of using them instead of selecting nodes directly based on the entropy source (block hash). At this time we know how to achieve uniform distribution and also current implementation uses same entropy source as may use Meresenne-Twister, ISAAC64, BBS or any other RNG. So from this point of view we don't see advantages to move to another implementation.

---

### Comment by @LenyKholodov

@jagerman After discussion with team of your idea about Mersenne-Twister usage for blockchain based list building we decided to accept it and rework supernodes selection with it. The main advantage of Mersenne-Twister is possibility to select more than 32 supernodes. We don't know now how many nodes we will select in prod environment. However, your approach is more flexible for such selection. Thank you very much again for your efforts. 

---

### Comment by @jagerman

> Thank you very much again for your efforts.

I am pleased to hear it and happy to help.  My apologies if discussion got a little overheated.

---

### Comment by @yidakee

Way to go team!

---

### Comment by @LenyKholodov

> > Thank you very much again for your efforts.
> 
> I am pleased to hear it and happy to help. My apologies if discussion got a little overheated.

No problem. We appreciate your help and participation. It's much better to find issues with implementation on this stage rather than in production. 

---

### Review by @jagerman [COMMENTED]



---

### Review by @jagerman [COMMENTED]



---

### Review by @jagerman [COMMENTED]



---

### Review by @jagerman [COMMENTED]



---

### Review by @mbg033 [APPROVED]



---

