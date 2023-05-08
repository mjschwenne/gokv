#!/usr/bin/env python3
import os
import subprocess
import argparse
import json
import time

def do_quiet(cmd):
    subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True).communicate()

def get_thruput(other_client_machines, threads, reads, recordcount, runtime, cooldown, warmup):
    benchcmd = f"./bench-ops.py {threads} --runtime {runtime} --reads {reads} --recordcount {recordcount} --warmup {warmup} --cooldown {cooldown} --outfile /tmp/bench.out 2>/tmp/bench.err 1>/tmp/bench.err"

    # start bench-ops.py on other machines
    for m in other_client_machines:
        sshcmd = f"""ssh upamanyu@node{m} <<ENDSSH
        killall go-ycsb 2>/dev/null;
        cd /users/upamanyu/gokv/simplepb/bench/;
        nohup {benchcmd} &
ENDSSH
        """
        do_quiet(sshcmd)

    # start local bench-ops.py; relying on the other one still warming up for this
    # to not mess up the numbers
    do_quiet(benchcmd)

    bench_data = []
    with open("/tmp/bench.out", "r") as f:
        bench_data += [json.loads(f.read())]

    time.sleep(5)
    # now, copy the output file from the other machine
    for m in other_client_machines:
        do_quiet(f"scp upamanyu@node{m}:/tmp/bench.out /tmp/bench{m}.out")
        with open(f"/tmp/bench{m}.out", "r") as f:
            bench_data += [json.loads(f.read())]

    return bench_data
