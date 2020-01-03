# -*- coding: utf-8 -*-
from fabric.api import run, cd, put


def runbg_using_dtach(cmd, sockname="dtach"):
    sh = './' + sockname + '-run.sh'
    run('echo \'%s > %s.log 2>&1\' > %s; chmod 755 %s' % (cmd,sockname,sh,sh))
    return run('dtach -n `mktemp -u /tmp/%s.XXXX` %s'  % (sockname,sh))

def runbg_using_tmux(cmd, key):
    run('tmux new -d -s %s ./%s' % (key,cmd))

def host_type():
    run('uname -s')

def package_copy():
    copy_dir = '/data/castis/cache/'
    put('*.tar.gz', copy_dir)

def deploy():
    with cd('/home/d7/cdn'):
        run('killall cache-server || true')
        put('../cache-server/cache-server', '/home/d7/cdn/cache-server')
        opt = ' -listen-addr 0.0.0.0:8080'
        opt += ' -origin origin.yml'
        opt += ' -bandwidth-limit-bps 100000000'
        opt += ' -cache-size 2097152' # 2MB
        runbg_using_dtach('./cache-server' + opt, 'cache-server')

# ngrinder-monitor가 돌아가고 있는 머신에서 동작
def aging(cpu_limit_percent=100, mem_limit_bytes=1024*1024*1024, base_test_id=200, repeat=1):
    with cd('/home/d7/cdn'):
        run('killall aging || true')
        run('killall monitor-custom.sh || true')

        put('aging/aging', '/home/d7/cdn/aging')
        monitor = '/home/d7/.ngrinder_agent/monitor/monitor-custom.sh'
        run('mkdir /home/d7/.ngrinder_agent/monitor || true')
        put('aging/monitor-custom.sh', monitor)
        run('chmod 755 ./aging')
        run('chmod 755 ' + monitor)

        runbg_using_dtach(monitor, 'monitor')

        agingopt = ' -cpu-threshold ' + str(cpu_limit_percent)
        agingopt += ' -mem-threshold ' + str(mem_limit_bytes)
        agingopt += ' -base-test-id ' + str(base_test_id)
        agingopt += ' -repeat ' + str(repeat)
        runbg_using_dtach('./aging' + agingopt, 'aging')

def deploy_and_aging(cpu_limit_percent=100, mem_limit_bytes=1024*1024*1024, base_test_id=200, repeat=1):
    deploy()
    aging(cpu_limit_percent, mem_limit_bytes, base_test_id, repeat)

def clear_aging():
    run('killall monitor-custom.sh || true')

def deploy_otm():
    with cd('/home/d7/cdn'):
        run('killall cache-server || true')
        run('killall cache-selector || true')
        run('killall http-lb || true')
        put('../build_go1.10/cache-server', '/home/d7/cdn/node1')
        put('../build_go1.10/cache-server', '/home/d7/cdn/node2')
        put('../build_go1.10/cache-selector', '/home/d7/cdn/esel')
        put('../build_go1.10/http-lb', '/home/d7/cdn/lb')
    with cd('/home/d7/cdn/node1'):
        runbg_using_tmux('./cache-server', 'cache-server1')
    with cd('/home/d7/cdn/node2'):
        runbg_using_tmux('./cache-server', 'cache-server2')
    with cd('/home/d7/cdn/esel'):
        runbg_using_tmux('./cache-selector', 'cache-selector')
    with cd('/home/d7/cdn/lb'):
        runbg_using_tmux('./http-lb', 'http-lb')
