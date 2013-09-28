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
package dude.morrildl.providence.gcm;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.net.MalformedURLException;
import java.security.KeyStoreException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.content.res.Resources.NotFoundException;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.net.Uri;
import android.os.Environment;
import android.provider.MediaStore;
import android.util.Log;

import com.android.volley.Request.Method;
import com.android.volley.Response.Listener;
import com.google.android.gcm.GCMBaseIntentService;

import dude.morrildl.providence.Stuff;
import dude.morrildl.providence.PanopticonActivity;
import dude.morrildl.providence.R;
import dude.morrildl.providence.db.OpenHelper;
import dude.morrildl.providence.volley.ByteArrayRequest;
import dude.morrildl.providence.volley.ByteArrayResponse;

public class GCMIntentService extends GCMBaseIntentService {
    // 10 hours
    private static final long MOTION_NOTIFICATION_THRESHOLD = 10 * 60 * 60 * 1000;

    public static final String SENDER_ID = "25235963451";

    public GCMIntentService() {
        super(SENDER_ID);
    }

    private void handleVbof(final Context context, String url) {
        Listener<ByteArrayResponse> listener = new Listener<ByteArrayResponse>() {
            @Override
            public void onResponse(ByteArrayResponse response) {
                try {
                    // Write the file to external storage
                    File f = Environment
                            .getExternalStoragePublicDirectory(Environment.DIRECTORY_PICTURES);
                    File vbofPath = new File(f.getCanonicalPath() + "/VBOF");
                    String fileId = "" + System.currentTimeMillis();
                    File vbofFile = new File(vbofPath.getCanonicalPath() + "/"
                            + fileId);
                    if (!vbofPath.exists()) {
                        vbofPath.mkdirs();
                    } else {
                        if (!vbofPath.isDirectory()) {
                            Log.e("GCMIntentService.doInBackground",
                                    "VBOF path not a directory: "
                                            + vbofPath.getCanonicalPath());
                            return;
                        }
                    }
                    FileOutputStream outputStream = new FileOutputStream(
                            vbofFile);
                    outputStream.write(response.bytes);
                    outputStream.close();

                    /*
                     * tell the MediaProvider about the new image
                     */
                    String imageTitle = response.headers.get("X-Image-Title");
                    imageTitle = imageTitle == null ? "" : imageTitle;
                    ContentValues values = new ContentValues(7);
                    if (imageTitle != null && !"".equals(imageTitle)) {
                        values.put(MediaStore.Images.Media.TITLE, imageTitle);
                        values.put(MediaStore.Images.Media.DESCRIPTION,
                                imageTitle);
                    } else {
                        values.put(MediaStore.Images.Media.TITLE, fileId);
                    }
                    String mimeType = response.headers.get("Content-Type");
                    if (mimeType != null && !"".equals(mimeType)) {
                        values.put(MediaStore.Images.Media.MIME_TYPE, mimeType);
                    }
                    values.put(MediaStore.Images.Media.BUCKET_ID,
                            vbofFile.hashCode());
                    values.put("_data", vbofFile.getCanonicalPath());
                    Uri target = getContentResolver().insert(
                            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
                            values);

                    // fire off a view Intent for the newly-written URL
                    Intent i = new Intent(Intent.ACTION_VIEW);
                    i.setData(target);
                    PendingIntent pi = PendingIntent.getActivity(context, 43,
                            i, 0);
                    Notification n = (new Notification.Builder(context))
                            .setContentTitle(
                                    context.getResources().getString(
                                            R.string.vbof_notif))
                            .setContentText(imageTitle).setContentIntent(pi)
                            .setSmallIcon(R.drawable.ic_stat_event)
                            .setAutoCancel(true).build();
                    ((NotificationManager) context
                            .getSystemService(Context.NOTIFICATION_SERVICE))
                            .notify(43, n);
                } catch (MalformedURLException e) {
                    Log.e("FetchVbofTask.doInBackground", "URL error", e);
                } catch (NotFoundException e) {
                    Log.e("FetchVbofTask.doInBackground", "URL error", e);
                } catch (IOException e) {
                    Log.e("FetchVbofTask.doInBackground", "transmission error",
                            e);
                } catch (ClassCastException e) {
                    Log.w("FetchVbofTask.doInBackground",
                            "did server send us the wrong kind of URL?", e);
                }
            }
        };

        ByteArrayRequest r = new ByteArrayRequest(Method.GET, url, listener,
                null);
        try {
            Stuff.getInstance(context).getRequestQueue().add(r);
        } catch (KeyStoreException e) {
        }
    }

    @Override
    protected void onError(Context context, String message) {
        Log.e("IntentService.onError", message);
    }

    @Override
    protected void onMessage(Context context, Intent intent) {
        // If the 'Url' extra is present, means it's a VBOF share
        String url = intent.getStringExtra("Url");
        if (url != null && !"".equals(url)) {
            handleVbof(context, url);
            return;
        }

        // Otherwise it's a security event
        boolean isMotion = (Integer.parseInt(intent
                .getStringExtra("SensorType")) == 2);
        boolean notifyOnMotion = false;
        boolean skipMotionUpdate = false;

        String which = intent.getStringExtra("WhichName");
        String ts = intent.getStringExtra("When");

        OpenHelper helper = new OpenHelper(context);
        SQLiteDatabase db = helper.getWritableDatabase();

        // insert all incoming events into the DB, even motion events
        ContentValues values = new ContentValues();
        values.put("which", which);
        values.put("type", intent.getStringExtra("SensorTypeName"));
        values.put("event", intent.getStringExtra("EventName"));
        values.put("ts", ts);
        values.put("eventid", intent.getStringExtra("EventId"));
        db.beginTransaction();
        db.insert("events", null, values);

        // filter out motion events, unless it's been a long time since the last
        if (isMotion) {
            Cursor c = db.query("last_motion", new String[] { "ts" },
                    "which = ?", new String[] { which }, null, null, null);
            if (c.moveToFirst()) { // i.e. we have heard from this sensor before
                Calendar lastMotion = Calendar.getInstance();
                Calendar currentMotion = Calendar.getInstance();
                try {
                    SimpleDateFormat sdf = new SimpleDateFormat(
                            "yyyy'-'MM'-'dd'T'HH:mm:ss");
                    lastMotion.setTime(sdf.parse(c.getString(0)));
                    currentMotion.setTime(sdf.parse(ts));

                    if (currentMotion.after(lastMotion)) {
                        // only notify if it's been > threshold since last one
                        // Note that this applies ONLY to firing a notification;
                        // it's up to other UI to show or not show motion events
                        // as is appropriate in context
                        notifyOnMotion = ((currentMotion.getTimeInMillis() - lastMotion
                                .getTimeInMillis()) > MOTION_NOTIFICATION_THRESHOLD);
                    } else {
                        // it's possible to receive these message out of order;
                        // tell ourselves to keep the more recent message
                        skipMotionUpdate = true;
                    }
                } catch (ParseException e) {
                    Log.w("GCM onMessage", "malformed date");
                }
            }
            // update or insert the timestamp, unless message was out of order
            if (!skipMotionUpdate) {
                db.execSQL(OpenHelper.REPLACE_LAST_MOTION, new Object[] {
                        which, ts });
            }
        }

        db.setTransactionSuccessful();
        db.endTransaction();
        db.close();

        if (!isMotion || notifyOnMotion) {
            Intent i = new Intent(context, PanopticonActivity.class);
            PendingIntent pi = PendingIntent.getActivity(context, 42, i, 0);
            Notification n = (new Notification.Builder(context))
                    .setContentTitle(
                            intent.getStringExtra("WhichName") + " "
                                    + intent.getStringExtra("EventName"))
                    .setContentIntent(pi)
                    .setSmallIcon(R.drawable.ic_stat_event).setAutoCancel(true)
                    .build();
            ((NotificationManager) context
                    .getSystemService(Context.NOTIFICATION_SERVICE)).notify(42,
                    n);
        }
    }

    @Override
    protected void onRegistered(Context context, String regId) {
        context.getSharedPreferences("main", 0).edit()
                .putString("regid", regId).commit();
    }

    @Override
    protected void onUnregistered(Context context, String regId) {
        context.getSharedPreferences("main", 0).edit().remove("regid").commit();
    }
}
