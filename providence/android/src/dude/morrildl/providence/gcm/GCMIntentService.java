package dude.morrildl.providence.gcm;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.util.Log;

import com.google.android.gcm.GCMBaseIntentService;

import dude.morrildl.providence.PanopticonActivity;
import dude.morrildl.providence.R;
import dude.morrildl.providence.panopticon.OpenHelper;

public class GCMIntentService extends GCMBaseIntentService {
	// 4 hours
	private static final long MOTION_NOTIFICATION_THRESHOLD = 4 * 60 * 60 * 1000;

	public static final String SENDER_ID = "25235963451";

	public GCMIntentService() {
		super(SENDER_ID);
	}

	@Override
	protected void onError(Context context, String message) {
		Log.e("IntentService.onError", message);
	}

	@Override
	protected void onMessage(Context context, Intent intent) {
		boolean isMotion = (Integer.parseInt(intent.getStringExtra("SensorType")) == 2);
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
					.getNotification();
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