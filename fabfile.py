from fabric.api import run, local, env, cd


env.hosts = ['root@msk1.cydev.ru:122']
root = '/src/poputchiki/'


def update():
    ver = local('git rev-parse HEAD', capture=True)
    with cd(root):
        remote_ver = run('git rev-parse HEAD')
        if ver == remote_ver:
            print('production already updated')
            return
        print('updating production to version %s' % ver)
        run('git reset --hard')
        run('git pull origin master')
        run('sed "s/VERSION/%s/g" Dockerfile.template > Dockerfile' % ver)
        run('docker build -t cydev/kafe .')
        run('docker stop kafe')
        run('docker rm kafe')
        run('~/poputchiki.sh')
        run('docker restart nginx')