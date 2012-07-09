package dude.morrildl.lifer.gcm;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;

import android.app.Activity;
import android.content.Context;
import android.os.AsyncTask;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;

import com.google.android.gcm.GCMRegistrar;

import dude.morrildl.lifer.R;

public class LiferActivity extends Activity {
	/** Called when the activity is first created. */
	@Override
	public void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);
		setContentView(R.layout.main);

		GCMRegistrar.checkDevice(this);
		GCMRegistrar.checkManifest(this);
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId == null || "".equals(regId)) {
			GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
		}
	}

	@Override
	public void onResume() {
		super.onResume();
		String regId = GCMRegistrar.getRegistrationId(this);
		if (regId != null && !"".equals(regId)) {
			if (!GCMRegistrar.isRegisteredOnServer(this)) {
				new ServerRegisterTask(this).execute(regId);
			}
		}

		Bundle b = getIntent().getExtras();
		if (b != null) {
			StringBuilder sb = new StringBuilder();
			sb.append(b.getString("WhichName"));
			String s = b.getString("SensorTypeName");
			if (s != null && !"".equals(s)) {
				sb.append(" (a ").append(s).append(")");
			}
			sb.append(" reported '").append(b.getString("EventName"))
					.append("' at ").append(b.getString("When"));
			((TextView) findViewById(R.id.main_text)).setText(sb.toString());
		} else {
			Log.e("booga booga wtf", "no extras");
		}
	}

	static class ServerRegisterTask extends AsyncTask<String, Integer, String> {
		private static final String SERVER_URL = "http://compatriot:4280/regid";
		private Context context;

		@SuppressWarnings("unused")
		private ServerRegisterTask() {
		}

		public ServerRegisterTask(Context context) {
			this.context = context;
		}

		@Override
		protected String doInBackground(String... regIds) {
			String regId = regIds[0];
			URL url;
			try {
				url = new URL(SERVER_URL);
			} catch (MalformedURLException e) {
				Log.e("doInBackground", "URL error", e);
				return null;
			}
			HttpURLConnection cxn = null;
			try {
				cxn = (HttpURLConnection) url.openConnection();
				cxn.setDoInput(true);
				cxn.setRequestMethod("POST");
				OutputStream os = cxn.getOutputStream();
				os.write(regId.getBytes());
				os.close();
				cxn.getInputStream().close();
			} catch (IOException e) {
				Log.e("doInBackground", "transmission error", e);
				return null;
			} finally {
				if (cxn != null)
					cxn.disconnect();
			}
			return regId;
		}

		@Override
		protected void onPostExecute(String result) {
			Log.e("onPostExecute", "null result :(");
			if (result != null)
				GCMRegistrar.setRegisteredOnServer(context, true);
		}
	}
}