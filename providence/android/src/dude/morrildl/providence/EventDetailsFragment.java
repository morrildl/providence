/* Copyright © 2012 Dan Morrill
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
package dude.morrildl.providence;

import java.io.IOException;
import java.io.StringReader;
import java.security.KeyStoreException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;

import android.app.Fragment;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.graphics.Bitmap;
import android.graphics.Bitmap.Config;
import android.os.AsyncTask;
import android.os.Bundle;
import android.util.JsonReader;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageView;
import android.widget.TextView;

import com.android.volley.Request.Method;
import com.android.volley.RequestQueue;
import com.android.volley.Response.ErrorListener;
import com.android.volley.Response.Listener;
import com.android.volley.VolleyError;

import dude.morrildl.providence.db.OpenHelper;
import dude.morrildl.providence.volley.ImageRequest;
import dude.morrildl.providence.volley.StringRequest;

public class EventDetailsFragment extends Fragment {
    private static final String TAG = "EventDetailsFragment";
    private long rowId;
    private Stuff stuff;

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        try {
            stuff = Stuff.getInstance(getActivity());
        } catch (KeyStoreException e) {
            Log.e(TAG, "error getting a handle to Network utility", e);
        }
        return inflater.inflate(R.layout.event_details_fragment, container, false);
    }

    @Override
    public void onHiddenChanged(boolean hidden) {
        super.onHiddenChanged(hidden);
        if (!hidden) {
            OpenHelper helper = new OpenHelper(this.getActivity());
            SQLiteDatabase db = helper.getReadableDatabase();
            Cursor c = db.query("events",
                    new String[] { "which", "type", "event", "ts", "eventid" }, "rowid = ?",
                    new String[] { Long.toString(rowId) }, null, null, null);

            StringBuilder sb = new StringBuilder();
            final String eventId;
            if (c.moveToFirst()) {
                eventId = c.getString(4);
                sb.append(c.getString(0));
                sb.append(" reported '").append(c.getString(2)).append("' ");

                try {
                    SimpleDateFormat sdf = new SimpleDateFormat("yyyy'-'MM'-'dd'T'HH:mm:ss");
                    java.util.Date parsedTs = sdf.parse(c.getString(3));

                    sdf = new SimpleDateFormat("EEE, dd MMMMMMMMM yyyy 'at' KK:mm:ssa");
                    sb.append("on ").append(sdf.format(parsedTs));
                } catch (ParseException e) {
                }
                ((TextView) getActivity().findViewById(R.id.event_details_fragment_main_text))
                        .setText(sb.toString());
            } else {
                ((TextView) getActivity().findViewById(R.id.event_details_fragment_main_text))
                        .setText(R.string.latest_event_default_text);
                return;
            }

            new AsyncTask<String, Integer, String>() {
                @Override
                protected String doInBackground(String... params) {
                    String token = "";
                    try {
                        token = stuff.fetchAuthToken();
                    } catch (OAuthException e) {
                        Log.e(TAG, "error fetching OAuth token", e);
                    }
                    return token;

                }

                @Override
                protected void onPostExecute(final String token) {
                    if ("".equals(token) || token == null) {
                        Log.w(TAG, "skipping volley request due to missing token");
                        return;
                    }

                    final RequestQueue q = stuff.getRequestQueue();
                    StringRequest sr = new StringRequest(Method.POST, stuff.getPhotosBase(),
                            new Listener<String>() {
                                @Override
                                public void onResponse(String response) {
                                    Log.e("trolololololololololololol", response);
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
                                    } catch (IOException e) {
                                    }
                                    Log.e("trolololololololololololol", urls.toString());
                                    for (String url : urls) {
                                        url = url.trim();
                                        ImageRequest ir = new ImageRequest(url,
                                                new Listener<Bitmap>() {
                                                    @Override
                                                    public void onResponse(Bitmap response) {
                                                        response = Bitmap.createScaledBitmap(
                                                                response, 200, 150, false);
                                                        ((ImageView) getActivity().findViewById(
                                                                R.id.the_image))
                                                                .setImageBitmap(response);
                                                    }
                                                }, 0, 0, Config.ARGB_4444, new ErrorListener() {
                                                    @Override
                                                    public void onErrorResponse(VolleyError error) {
                                                        String code = error.networkResponse != null ? ""
                                                                + error.networkResponse.statusCode
                                                                : "";
                                                        Log.w(TAG, "volley responded with error "
                                                                + code, error.getCause());
                                                    }
                                                });
                                        ir.setHeader("X-OAuth-JWT", token);
                                        ir.setShouldCache(true);
                                        q.add(ir);
                                    }
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
                    sr.setBody(eventId);
                    q.add(sr);
                }
            }.execute("");
        }
    }

    public void setId(long id) {
        this.rowId = id;
    }
}
