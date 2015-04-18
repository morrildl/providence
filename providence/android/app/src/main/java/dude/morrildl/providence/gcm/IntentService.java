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

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.util.Log;

import com.google.android.gms.gcm.GoogleCloudMessaging;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;

import dude.morrildl.providence.PanopticonActivity;
import dude.morrildl.providence.R;
import dude.morrildl.providence.db.OpenHelper;

public class IntentService extends android.app.IntentService {
    public static final String SENDER_ID = "25235963451";
    public static final String SERVICE_NAME = "GCM_HANDLER";
    // 10 hours
    private static final long MOTION_NOTIFICATION_THRESHOLD = 10 * 60 * 60 * 1000;

    public IntentService() {
        super(SERVICE_NAME);
    }

    @Override
    protected void onHandleIntent(Intent intent) {
        String msgType = (GoogleCloudMessaging.getInstance(this)).getMessageType(intent);
        Log.e("booga booga booga", "flex " + msgType);
        for (String s : intent.getExtras().keySet()) {
            Log.e("booga booga booga", "flex " + s);
        }
        if (msgType == null || "".equals(msgType))
            return;

        switch (msgType) {
            case GoogleCloudMessaging.MESSAGE_TYPE_MESSAGE:
                break;
            case GoogleCloudMessaging.MESSAGE_TYPE_SEND_ERROR:
            case GoogleCloudMessaging.MESSAGE_TYPE_DELETED:
                Log.e("GCM_HANDLER", "Non-message received: " + intent.getExtras().toString());
            default:
                return;
        }

        boolean isMotion = (Integer.parseInt(intent.getStringExtra("SensorType")) == 2);
        boolean notifyOnMotion = false;
        boolean skipMotionUpdate = false;

        String which = intent.getStringExtra("WhichName");
        String ts = intent.getStringExtra("When");

        OpenHelper helper = new OpenHelper(this);
        SQLiteDatabase db = helper.getWritableDatabase();

        // insert all incoming events into the DB, even motion events
        ContentValues values = new ContentValues();
        values.put("which", which);
        values.put("type", intent.getStringExtra("SensorTypeName"));
        values.put("event", intent.getStringExtra("EventName"));
        values.put("ts", ts);
        values.put("eventid", intent.getStringExtra("EventId"));
        db.insert("events", null, values);

        db.beginTransaction();
        // filter out motion events, unless it's been a long time since the last
        if (isMotion) {
            Cursor c = db.query("last_motion", new String[]{"ts"}, "which = ?", new String[]{which},
                                null, null, null);
            if (c.moveToFirst()) { // i.e. we have heard from this sensor before
                Calendar lastMotion = Calendar.getInstance();
                Calendar currentMotion = Calendar.getInstance();
                try {
                    SimpleDateFormat sdf = new SimpleDateFormat("yyyy'-'MM'-'dd'T'HH:mm:ss");
                    lastMotion.setTime(sdf.parse(c.getString(0)));
                    currentMotion.setTime(sdf.parse(ts));

                    if (currentMotion.after(lastMotion)) {
                        // only notify if it's been > threshold since last one
                        // Note that this applies ONLY to firing a notification;
                        // it's up to other UI to show or not show motion events
                        // as is appropriate in context
                        notifyOnMotion = ((currentMotion.getTimeInMillis() - lastMotion.getTimeInMillis()) > MOTION_NOTIFICATION_THRESHOLD);
                    } else {
                        // it's possible to receive these message out of order;
                        // tell ourselves to keep the more recent message
                        skipMotionUpdate = true;
                    }
                } catch (ParseException e) {
                    Log.w("GCM onMessage", "malformed date");
                }
            }
            c.close();
            // update or insert the timestamp, unless message was out of order
            if (!skipMotionUpdate) {
                db.execSQL(OpenHelper.REPLACE_LAST_MOTION, new Object[]{which, ts});
            }
        }

        db.setTransactionSuccessful();
        db.endTransaction();
        db.close();

        if (!isMotion || notifyOnMotion) {
            Intent i = new Intent(this, PanopticonActivity.class);
            i.setFlags(
                    Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK | Intent.FLAG_ACTIVITY_TASK_ON_HOME);
            PendingIntent pi = PendingIntent.getActivity(this, 42, i,
                                                         PendingIntent.FLAG_UPDATE_CURRENT);
            Notification n = (new Notification.Builder(this)).setContentTitle(
                    intent.getStringExtra("WhichName") + " " + intent.getStringExtra(
                            "EventName")).setContentIntent(pi).setSmallIcon(
                    R.drawable.ic_stat_event).setAutoCancel(true).build();
            ((NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE)).notify(42, n);
        }
    }
}
