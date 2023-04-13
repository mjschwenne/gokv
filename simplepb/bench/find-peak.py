#!/usr/bin/env -S python3 -u

import os
from os import system as do
from parseycsb import *
import sys
import argparse
import json
from bench_ops_multiclient import *

parser = argparse.ArgumentParser()
parser.add_argument('--onlypeak',
                    help='if this flag is set, then this only prints the peak throughput, not all the intermediate throughputs',
                    action='store_true')
parser.add_argument("--reads",
                    help="percentage of ops that are reads (between 0.0 and 1.0)",
                    required=False,
                    default=0.0)
parser.add_argument('--benchcmd',
                    help='The command to run to benchmark a single server (should be either the one for GroveKV or for redis)',
                    default='./bench-ops-multiclient.py')
args = parser.parse_args()

threadcounts = [50, 100, 150, 200, 250] + [200 * (i + 2) for i in range(12)]

runtime = 10
warmuptime = 0
fullwarmuptime = 20
fullruntime = 120
recordcount = 1000

if not args.onlypeak:
    print("threads, throughput, avglatency")

highestThreads = 0
highestThruput = 0
highestThruputLatency = 0

for threads in threadcounts:
    ms = get_thruput(threads, args.reads, recordcount, 10, warmuptime)
    thruput = 0.0
    latency = 0.0
    for m in ms:
        for optype in m["lts"]:
            thruput += m["lts"][optype]["thruput"]
            latency += m["lts"][optype]["avg_latency"]

    latency = latency/len(ms)

    if not args.onlypeak:
        print(f"{threads}, {thruput}, {latency}")

    if thruput > highestThruput:
        highestThruput = thruput
        highestThreads = threads
        highestThruputLatency = latency
    if 1.10 * thruput < highestThruput:
        break # don't bother trying them all


if args.onlypeak:
    print(f"{highestThruput}, {highestThreads}, {highestThruputLatency}")
else:
    print(f"# Highest throughput {highestThruput} at {highestThreads} threads")

sys.exit(0) # Don't bother with running for longer for now

# Run the nthread that gave the highest throughput for longer
benchoutput = os.popen(f"{args.benchcmd} {highestThreads} --warmup {fullwarmuptime}", 'r', 100)
ret = ''
for line in benchoutput:
    if line.find('Takes(s): {0}.'.format(fullruntime)) != -1:
        ret = line
        benchoutput.close()
        break
time, ops, latency = (parse_ycsb_output_totalops(ret))
thruput = ops/time

print(f"# sustained throughput {thruput} at {highestThreads} threads")
