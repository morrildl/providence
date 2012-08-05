package dude.morrildl.providence.gcm;

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.database.sqlite.SQLiteDatabase;
import android.util.Log;

import com.google.android.gcm.GCMBaseIntentService;

import dude.morrildl.providence.PanopticonActivity;
import dude.morrildl.providence.R;
import dude.morrildl.providence.panopticon.OpenHelper;

public class GCMIntentService extends GCMBaseIntentService {
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
		OpenHelper helper = new OpenHelper(context);
		SQLiteDatabase db = helper.getWritableDatabase();
		db.beginTransaction();
		ContentValues values = new ContentValues();
		values.put("which", intent.getStringExtra("WhichName"));
		values.put("type", intent.getStringExtra("SensorTypeName"));
		values.put("event", intent.getStringExtra("EventName"));
		values.put("ts", intent.getStringExtra("When"));
		db.insert("events", null, values);
		db.execSQL(OpenHelper.GC);
		db.setTransactionSuccessful();
		db.endTransaction();
		
		Intent i = new Intent(context, PanopticonActivity.class);
		PendingIntent pi = PendingIntent.getActivity(context, 42, i, 0);
		Notification n = (new Notification.Builder(context))
				.setContentTitle("Event fired").setContentIntent(pi)
				.setSmallIcon(R.drawable.ic_stat_event).setAutoCancel(true)
				.getNotification();
		((NotificationManager) context
				.getSystemService(Context.NOTIFICATION_SERVICE)).notify(42, n);
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
