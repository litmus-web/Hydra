import pathlib
from setuptools import setup, find_packages

HERE = pathlib.Path(__file__).parent

README = (HERE / "README.md").read_text()

setup(
    author="Harrison Burt",
    author_email="hburt2003@gmail.com",

    name="hydra-client",
    version="0.0.1",

    description="A mythical HTTP server that accelerates your existing applications.",
    long_description=README,
    long_description_content_type="text/markdown",

    url="https://github.com/Project-Dream-Weaver/Hydra",
    packages=["hydra_client"],
    download_url="https://github.com/Project-Dream-Weaver/Hydra/archive/0.0.1.tar.gz",
    install_requires=[
        "aiohttp",
    ],

    license="Apache-2.0",
    classifiers=[
        "License :: OSI Approved :: Apache License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
    ],

)
