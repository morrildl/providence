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

import java.io.IOException;
import java.io.OutputStream;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyStoreException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;

import javax.net.ssl.HttpsURLConnection;

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
import android.os.AsyncTask;
import android.provider.MediaStore;
import android.util.Log;

import com.google.android.gcm.GCMBaseIntentService;

import dude.morrildl.providence.PanopticonActivity;
import dude.morrildl.providence.R;
import dude.morrildl.providence.panopticon.OpenHelper;
import dude.morrildl.providence.util.Network;

public class GCMIntentService extends GCMBaseIntentService {
	class FetchVbofTask extends AsyncTask<String, Integer, Boolean> {
		private Context context;

		public FetchVbofTask(Context context) {
			this.context = context;
		}

		@Override
		protected Boolean doInBackground(String... params) {
			URL url;
			HttpsURLConnection cxn = null;
			try {
				Network networkUtil = Network.getInstance(context);
				url = new URL(params[0]);
				cxn = (HttpsURLConnection) url.openConnection();
				cxn.setSSLSocketFactory(networkUtil.getSslSocketFactory());
				cxn.setDoInput(true);
				cxn.setRequestMethod("GET");
				String mimeType = cxn.getContentType();
				String imageTitle = cxn.getHeaderField("X-Image-Title");
				byte[] bytes = networkUtil.readStream(cxn.getContentLength(),
						cxn.getInputStream());
				if (bytes == null || bytes.length == 0) {
					return false;
				}

				ContentValues values = new ContentValues(7);
				/*
				 * values.put(MediaStore.Images.Media.DISPLAY_NAME,
				 * "name of the picture");
				 * values.put(MediaStore.Images.Media.DESCRIPTION,
				 * "Some description");
				 * values.put(MediaStore.Images.Media.DATE_TAKEN
				 * ,System.currentTimeMillis());
				 */
				if (imageTitle != null && !"".equals(imageTitle)) {
					values.put(MediaStore.Images.Media.TITLE, imageTitle);
				}
				if (mimeType != null && !"".equals(mimeType)) {
					values.put(MediaStore.Images.Media.MIME_TYPE, mimeType);
				}
				values.put(MediaStore.Images.Media.BUCKET_ID, 31337);
				values.put(MediaStore.Images.Media.BUCKET_DISPLAY_NAME, "VBOF");
				/*
				Uri uri = getContentResolver().insert(
						MediaStore.Images.Media.EXTERNAL_CONTENT_URI, values);
				*/
				Uri uri = Uri.parse("file:///sdcard/VBOF/booga.jpg");
				OutputStream outStream = getContentResolver().openOutputStream(
						uri);
				outStream.write(bytes);
				outStream.close();

				Intent i = new Intent(Intent.ACTION_VIEW);
				i.setData(uri);
				PendingIntent pi = PendingIntent.getActivity(context, 43, i, 0);
				Notification n = (new Notification.Builder(context))
						.setContentTitle(
								context.getResources().getString(
										R.string.vbof_notif))
						.setContentIntent(pi)
						.setSmallIcon(R.drawable.ic_stat_event)
						.setAutoCancel(true).getNotification();
				((NotificationManager) context
						.getSystemService(Context.NOTIFICATION_SERVICE))
						.notify(43, n);
			} catch (MalformedURLException e) {
				Log.e("FetchVbofTask.doInBackground", "URL error", e);
			} catch (NotFoundException e) {
				Log.e("FetchVbofTask.doInBackground", "URL error", e);
			} catch (IOException e) {
				Log.e("FetchVbofTask.doInBackground", "transmission error", e);
			} catch (KeyStoreException e) {
				Log.e("FetchVbofTask.doInBackground", "error setting up SSL", e);
			} catch (ClassCastException e) {
				Log.w("FetchVbofTask.doInBackground",
						"did server send us the wrong kind of URL?", e);
			} finally {
				if (cxn != null)
					cxn.disconnect();
			}
			return false;
		}

		@Override
		protected void onPostExecute(Boolean success) {
		}
	}

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
		String url = intent.getStringExtra("Url");
		if (url != null && !"".equals(url)) {
			new FetchVbofTask(this).execute(url);
			return;
		}

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
