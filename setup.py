from distutils.core import setup

setup(
    name='sandman',
    packages=['sandman'],
    version='0.1.0',
    license='MIT',
    description='A warp speed WSGI & ASGI web server powered by Rust',
    long_description='A http server written in Rust that binds to Python workers accelerating Python to warp '
                     'speed. A Production grade NGINX type server.',
    author='Harrison Burt',
    author_email='hburt2003@gmail.com',
    url='https://github.com/Project-Dream-Weaver/Sandman',
    download_url='https://github.com/Project-Dream-Weaver/Sandman/archive/0.1.0.tar.gz',
    keywords=[
        'web',
        'asgi',
        'webserver',
        'rust',
        'asgi',
        'asyncio',
        'http-server'
    ],
    install_requires=[
        'aiohttp',
        'pyuv',
        'websocket-client-py3',
    ],
    classifiers=[
        'Development Status :: 3 - Alpha',
        'Intended Audience :: Developers',
        'Topic :: Software Development :: Build Tools',
        'License :: OSI Approved :: MIT License',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
    ],
)