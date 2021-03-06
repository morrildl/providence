/* Copyright © 2013 Dan Morrill
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package dude.morrildl.providence.view;

import java.io.IOException;
import java.io.StringReader;
import java.util.ArrayList;

import android.content.Context;
import android.os.AsyncTask;
import android.util.JsonReader;
import android.util.Log;

import com.android.volley.Request.Method;
import com.android.volley.RequestQueue;
import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;
import com.android.volley.VolleyError;

import dude.morrildl.providence.Config;
import dude.morrildl.providence.NetworkHelper;
import dude.morrildl.providence.OAuthException;
import dude.morrildl.providence.volley.StringRequest;

/**
 * An AsyncTask that fetches from the server a list of photo URLs for a given
 * event ID. Those URLs are then passed to a target EventSummaryView to
 * individually load and display.
 */
class PhotoAsyncTask extends AsyncTask<String, Integer, String> {
    private static final String TAG = "PhotoAsyncTask";
    private final EventSummaryView esv;
    private final String eventId;
    private final Context context;

    public PhotoAsyncTask(Context context, EventSummaryView esv, String eventId) {
        this.esv = esv;
        this.eventId = eventId;
        this.context = context;
    }

    @Override
    protected String doInBackground(String... params) {
        NetworkHelper netHelper = NetworkHelper.getInstance(context);
        final String token;
        try {
            token = netHelper.fetchAuthToken();
        } catch (OAuthException e) {
            Log.w("PhotoAsyncTask.doInBackground",
                    "unable to fetch auth token", e);
            return null;
        }
        final RequestQueue q = netHelper.getRequestQueue();
        StringRequest sr = new StringRequest(Method.POST, Config.getInstance(
                context).getPhotoUrlBase()
                + eventId, new Listener<String>() {
            @Override
            public void onResponse(String response) {
                // the response is actually a JSON array, so parse it
                JsonReader reader = new JsonReader(new StringReader(response));
                ArrayList<String> urls = new ArrayList<String>();
                try {
                    reader.beginObject();
                    while (reader.hasNext()) {
                        reader.nextName(); // should be only one
                        reader.beginArray();
                        while (reader.hasNext()) {
                            urls.add(reader.nextString());
                        }
                        reader.endArray();
                    }
                    reader.endObject();
                    reader.close();
                } catch (IOException e) {
                    Log.w(TAG, "IOException parsing photo list response", e);
                    return;
                }

                // okay, hand it off to the View.
                esv.loadImages(token, urls);

                /*
                 * This is a bit awkward since esv is itself going to use Volley
                 * to make more HTTPS requests. We could do it here just as well
                 * and keep such fetch logic out of the View. Unfortunately,
                 * since the View can be recycled, we end up with two different
                 * threads racing for access to the View: the ListView framework
                 * asynchronously recycles these instances, and we are
                 * asynchronously pushing images to them. The result is
                 * user-visible thrashing (and even errors and crashes) in the
                 * images for the event. So instead we push the fetch-images
                 * logic into the View itself, since it knows best when it is
                 * being recycled and thus needs to cancel Volley requests. We
                 * could also do a separate bookkeeping structure for this, but
                 * that seems like overkill.
                 */
            }
        }, new ErrorListener() {
            @Override
            public void onErrorResponse(VolleyError error) {
                String code = error.networkResponse != null ? ""
                        + error.networkResponse.statusCode : "";
                Log.w(TAG, "volley responded with error " + code,
                        error.getCause());
            }
        });
        sr.setShouldCache(true);
        sr.setHeader("X-OAuth-JWT", token);
        q.add(sr);
        return null;
    }
}