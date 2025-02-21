#! /usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2020-2023 Alibaba Group Holding Limited.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import contextlib
import logging
import os
import sys
import traceback

from .version import __version__

__doc__ = """
Vineyard - an in-memory immutable data manager. (Project under CNCF)
====================================================================

Vineyard (v6d) is an in-memory immutable data manager that provides
out-of-the-box high-level abstraction and zero-copy in-memory
sharing for distributed data in big data tasks, such as graph analytics
(e.g., GraphScope), numerical computing (e.g., Mars), and machine learning.
"""

# pylint: disable=import-outside-toplevel,wrong-import-position

logger = logging.getLogger('vineyard')


@contextlib.contextmanager
def envvars(key: str, value: str = None, append: bool = False) -> None:
    """Create a context with specified environment variables set.

    It is useful for setting the :code`VINEYARD_IPC_SOCKET` environment
    variable to obtain a proper default vineyard client.

    This context macro can be used as

    .. code:: python

        with environment('KEY'):
            # env :code:`KEY` will be set to None.

        with environment('KEY', 'value'):
            # env :code:`KEY` will be set as :code:`value`.

        with environment({'KEY1': None, 'KEY2': 'value2'}):
            # env :code:`KEY1` will be set as None and :code:`KEY2` will
            # be set as :code:`value2`.
    """
    items = key
    if isinstance(key, str):
        items = {key: value}
    original_items = dict()
    for k, v in items.items():
        original_items[k] = os.environ.get(k, None)
        if append and original_items[k] is not None:
            os.environ[k] = original_items[k] + ':' + v
        else:
            os.environ[k] = v

    yield os.environ

    for k, v in original_items.items():
        if v is not None:
            os.environ[k] = v
        else:
            del os.environ[k]


def _init_global_context():
    import os as _dl_flags  # pylint: disable=reimported

    if sys.platform == 'linux':
        registry = os.path.join(
            os.path.dirname(__file__), 'libvineyard_internal_registry.so'
        )
    elif sys.platform == 'darwin':
        registry = os.path.join(
            os.path.dirname(__file__), 'libvineyard_internal_registry.dylib'
        )
    else:
        raise RuntimeError("Unsupported platform: %s" % sys.platform)

    ctx = {'__VINEYARD_INTERNAL_REGISTRY': registry}

    if os.environ.get('VINEYARD_DEVELOP', None) is None:
        with envvars(ctx):  # n.b., no append
            from . import _C
        return

    if not hasattr(_dl_flags, 'RTLD_GLOBAL') or not hasattr(_dl_flags, 'RTLD_LAZY'):
        try:
            # next try if DLFCN exists
            import DLFCN as _dl_flags  # noqa: N811
        except ImportError:
            _dl_flags = None

    if _dl_flags is not None:
        old_flags = sys.getdlopenflags()

        # import the extension module
        sys.setdlopenflags(_dl_flags.RTLD_GLOBAL | _dl_flags.RTLD_LAZY)
        with envvars(ctx):  # n.b., no append
            from . import _C  # noqa: F811

        # See Note [Import pyarrow before _C]
        sys.setdlopenflags(_dl_flags.RTLD_GLOBAL | _dl_flags.RTLD_LAZY)
        import pyarrow

        del pyarrow

        # restore
        sys.setdlopenflags(old_flags)


_init_global_context()
del _init_global_context


from . import core
from . import data
from . import deploy
from . import io
from . import launcher
from . import shared_memory
from ._C import ArrowErrorException
from ._C import AssertionFailedException
from ._C import Blob
from ._C import BlobBuilder
from ._C import ConnectionErrorException
from ._C import ConnectionFailedException
from ._C import EndOfFileException
from ._C import EtcdErrorException
from ._C import InstanceStatus
from ._C import InvalidException
from ._C import InvalidStreamStateException
from ._C import IOErrorException
from ._C import IPCClient
from ._C import KeyErrorException
from ._C import MetaTreeInvalidException
from ._C import MetaTreeLinkInvalidException
from ._C import MetaTreeNameInvalidException
from ._C import MetaTreeNameNotExistsException
from ._C import MetaTreeSubtreeNotExistsException
from ._C import MetaTreeTypeInvalidException
from ._C import MetaTreeTypeNotExistsException
from ._C import NotEnoughMemoryException
from ._C import NotImplementedException
from ._C import Object
from ._C import ObjectBuilder
from ._C import ObjectExistsException
from ._C import ObjectID
from ._C import ObjectMeta
from ._C import ObjectName
from ._C import ObjectNotExistsException
from ._C import ObjectNotSealedException
from ._C import ObjectSealedException
from ._C import RemoteBlob
from ._C import RemoteBlobBuilder
from ._C import RPCClient
from ._C import StreamDrainedException
from ._C import StreamFailedException
from ._C import TypeErrorException
from ._C import UnknownErrorException
from ._C import UserInputErrorException
from ._C import VineyardServerNotReadyException
from ._C import connect
from ._C import memory_copy
from .core import builder_context
from .core import default_builder_context
from .core import default_driver_context
from .core import default_resolver_context
from .core import driver_context
from .core import resolver_context
from .data import register_builtin_types
from .data.graph import Graph
from .deploy.local import get_current_client
from .deploy.local import get_current_socket
from .deploy.local import init
from .deploy.local import shutdown


def _init_vineyard_modules():  # noqa: C901
    """Resolve registered vineyard modules in the following order:

    * /etc/vineyard/config.py
    * {sys.prefix}/etc/vineyard/config.py
    * /usr/share/vineyard/01-xxx.py
    * /usr/local/share/vineyard/01-xxx.py
    * {sys.prefix}/share/vineyard/02-xxxx.py
    * $HOME/.vineyard/03-xxxxx.py

    Then import packages like vineyard.drivers.*:

    * vineyard.drivers.io
    """

    import glob
    import importlib.util
    import pkgutil
    import site
    import sysconfig

    def _import_module_from_file(filepath):
        filepath = os.path.expanduser(os.path.expandvars(filepath))
        if os.path.exists(filepath):
            try:
                spec = importlib.util.spec_from_file_location(
                    "vineyard._contrib", filepath
                )
                mod = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(mod)
            except Exception:  # pylint: disable=broad-except
                logger.debug("Failed to load %s", filepath, exc_info=True)

    def _import_module_from_qualified_name(module):
        try:
            importlib.import_module(module)
        except Exception:  # pylint: disable=broad-except
            logger.debug('Failed to load module %s', module, exc_info=True)

    _import_module_from_file('/etc/vineyard/config.py')
    _import_module_from_file(os.path.join(sys.prefix, '/etc/vineyard/config.py'))
    for filepath in glob.glob('/usr/share/vineyard/*-*.py'):
        _import_module_from_file(filepath)
    for filepath in glob.glob('/usr/local/share/vineyard/*-*.py'):
        _import_module_from_file(filepath)
    for filepath in glob.glob(os.path.join(sys.prefix, '/share/vineyard/*-*.py')):
        _import_module_from_file(filepath)
    for filepath in glob.glob(os.path.expanduser('$HOME/.vineyard/*-*.py')):
        _import_module_from_file(filepath)

    package_sites = set()
    pkg_sites = site.getsitepackages()
    if not isinstance(pkg_sites, (list, tuple)):
        pkg_sites = [pkg_sites]
    package_sites.update(pkg_sites)
    pkg_sites = site.getusersitepackages()
    if not isinstance(pkg_sites, (list, tuple)):
        pkg_sites = [pkg_sites]
    package_sites.update(pkg_sites)

    paths = sysconfig.get_paths()
    if 'purelib' in paths:
        package_sites.add(paths['purelib'])
    if 'platlib' in paths:
        package_sites.add(paths['purelib'])

    # add relative path
    package_sites.add(os.path.join(os.path.dirname(__file__), '..'))

    # dedup
    deduped = set()
    for pkg_site in package_sites:
        deduped.add(os.path.abspath(pkg_site))

    for pkg_site in deduped:
        for _, mod, _ in pkgutil.iter_modules(
            [os.path.join(pkg_site, 'vineyard', 'drivers')]
        ):
            _import_module_from_qualified_name('vineyard.drivers.%s' % mod)


try:
    _init_vineyard_modules()
except Exception:  # pylint: disable=broad-except
    pass
del _init_vineyard_modules
