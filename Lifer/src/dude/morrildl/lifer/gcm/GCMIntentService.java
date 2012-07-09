package dude.morrildl.lifer.gcm;

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.util.Log;

import com.google.android.gcm.GCMBaseIntentService;

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
		Intent i = new Intent(context, LiferActivity.class);
		i.putExtras(intent.getExtras());
		PendingIntent pi = PendingIntent.getActivity(context, 42, i,
				PendingIntent.FLAG_UPDATE_CURRENT);
		Notification n = (new Notification.Builder(context))
				.setContentTitle("Event fired").setContentIntent(pi)
				.setSmallIcon(android.R.drawable.ic_delete).setAutoCancel(true)
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
