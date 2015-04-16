/*
 * Copyright (C) 2011 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dude.morrildl.providence.volley;

import java.util.HashMap;

import com.android.volley.NetworkResponse;
import com.android.volley.Request;
import com.android.volley.Response;
import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;
import com.android.volley.toolbox.HttpHeaderParser;

public class ByteArrayRequest extends Request<ByteArrayResponse> {
    private final Listener<ByteArrayResponse> listener;

    public ByteArrayRequest(int method, String url, Listener<ByteArrayResponse> listener,
            ErrorListener errorListener) {
        super(method, url, errorListener);
        this.listener = listener;
    }

    @Override
    protected void deliverResponse(ByteArrayResponse response) {
        if (listener != null) {
            listener.onResponse(response);
        }
    }

    @Override
    protected Response<ByteArrayResponse> parseNetworkResponse(NetworkResponse response) {
        HashMap<String, String> headers = new HashMap<String, String>();
        headers.putAll(response.headers);
        return Response.success(new ByteArrayResponse(response.data, headers),
                HttpHeaderParser.parseCacheHeaders(response));
    }
}
