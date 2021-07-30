#! /usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2020-2021 Alibaba Group Holding Limited.
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

from vineyard._C import ObjectMeta
from vineyard.core.resolver import resolver_context
from vineyard.data.utils import from_json, to_json, build_numpy_buffer, normalize_dtype
from vineyard.data import tensor, dataframe, arrow

import pandas as pd
import pyarrow as pa
try:
    from pandas.core.internals.blocks import BlockPlacement, NumpyBlock as Block
except:
    BlockPlacement = None
    from pandas.core.internals.blocks import Block

from pandas.core.internals.managers import BlockManager
import numpy as np
import xgboost as xgb


def xgb_builder(client, value, builder, **kw):
    # TODO: build DMatrix to vineyard objects
    pass


def xgb_tensor_resolver(obj):
    array = tensor.numpy_ndarray_resolver(obj)
    return xgb.DMatrix(array)


def xgb_dataframe_resolver(obj, resolver, **kw):
    with resolver_context({'vineyard::Tensor': tensor.numpy_ndarray_resolver}) as ctx:
        df = dataframe.pandas_dataframe_resolver(obj, ctx)
        if 'label' in kw:
            label = df.pop(kw['label'])
            # data column can only be specified if label column is specified
            if 'data' in kw:
                df = np.stack(df[kw['data']].values)
            return xgb.DMatrix(df, label)
        return xgb.DMatrix(df)


def xgb_recordBatch_resolver(obj, resolver, **kw):
    rb = arrow.record_batch_resolver(obj, resolver)
    # FIXME to_pandas is not zero_copy guaranteed
    df = rb.to_pandas()
    if 'label' in kw:
        label = df.pop(kw['label'])
        return xgb.DMatrix(df, label)
    return xgb.DMatrix(df)


def xgb_table_resolver(obj, resolver, **kw):
    with resolver_context({'vineyard::RecordBatch': arrow.record_batch_resolver}) as ctx:
        tb = arrow.table_resolver(obj, ctx)
        # FIXME to_pandas is not zero_copy guaranteed
        df = tb.to_pandas()
        if 'label' in kw:
            label = df.pop(kw['label'])
            return xgb.DMatrix(df, label)
        return xgb.DMatrix(df)


def register_xgb_types(builder_ctx, resolver_ctx):
    if builder_ctx is not None:
        builder_ctx.register(xgb.DMatrix, xgb_builder)

    if resolver_ctx is not None:
        resolver_ctx.register('vineyard::Tensor', xgb_tensor_resolver)
        resolver_ctx.register('vineyard::DataFrame', xgb_dataframe_resolver)
        resolver_ctx.register('vineyard::RecordBatch', xgb_recordBatch_resolver)
        resolver_ctx.register('vineyard::Table', xgb_table_resolver)
