#!/usr/bin/env python3

from os import system as do
for i in range(8):
    do(f"""ssh node{str(i)} 'sudo ~/gokv/vrsm/bench/only-one-cpu.py'""")
