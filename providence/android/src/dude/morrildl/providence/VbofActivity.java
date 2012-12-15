package dude.morrildl.providence;

import java.io.File;
import java.io.IOException;
import java.io.OutputStream;
import java.net.MalformedURLException;
import java.net.URL;
import java.security.KeyStoreException;

import javax.net.ssl.HttpsURLConnection;

import android.app.Activity;
import android.content.Intent;
import android.content.res.Resources.NotFoundException;
import android.net.Uri;
import android.os.AsyncTask;
import android.os.Bundle;
import android.util.Log;
import dude.morrildl.providence.util.Network;

public class VbofActivity extends Activity {

	/**
	 * Handles the actual act of sending a VBOF share.
	 */
	class VbofSendTask extends AsyncTask<Intent, Integer, Boolean> {
		public VbofSendTask() {
		}

		@Override
		protected Boolean doInBackground(Intent... params) {
			if (params == null | params[0] == null) {
				Log.w("VbofSendTask.doInBackground", "sent no-op intent");
				return true;
			}
			
			// Pull out the info from the Intent we are to share
			Intent intent = params[0];
			String mimeType = intent.resolveType(getContentResolver());
			String subject = intent.getStringExtra(Intent.EXTRA_SUBJECT);
			Uri uri = intent.getParcelableExtra(Intent.EXTRA_STREAM);

			// load the bytes pointed to by the URI
			String scheme = uri.getScheme();
			byte[] bytes = null;
			if (scheme == "file") {
				bytes = networkUtil.readFile(new File(uri.getPath()));
			} else if ("http".equals(scheme) || "https".equals(scheme)) {
				bytes = networkUtil.readUrl(uri.toString());
			} else if ("content".equals(scheme)) {
				bytes = networkUtil.readProvider(uri.toString());
			} else {
				Log.w("VbofSendTask.doInBackground",
						"don't know how to handle scheme " + scheme);
			}
			if (bytes == null || bytes.length == 0) {
				Log.w("VbofSendTask.doInBackground", "got empty or null bytes");
				return false;
			}

			URL url;
			HttpsURLConnection cxn = null;
			try {
				// Connect to server and upload the image data & metadata
				url = networkUtil.urlForResource(R.raw.vbof_send_url, subject);
				cxn = (HttpsURLConnection) url.openConnection();
				cxn.setSSLSocketFactory(networkUtil.getSslSocketFactory());
				cxn.setDoInput(true);
				if (mimeType != null && !"".equals(mimeType)) {
					cxn.addRequestProperty("Content-Type", mimeType);
				}
				cxn.setRequestMethod("POST");
				OutputStream os = cxn.getOutputStream();
				os.write(bytes);
				os.close();
				return "OK".equals(new java.util.Scanner(cxn.getInputStream())
						.useDelimiter("\\A").next().trim());
			} catch (MalformedURLException e) {
				Log.e("doInBackground", "URL error", e);
			} catch (NotFoundException e) {
				Log.e("doInBackground", "URL error", e);
			} catch (IOException e) {
				Log.e("doInBackground", "transmission error", e);
			} finally {
				if (cxn != null)
					cxn.disconnect();
			}

			return false;
		}

		@Override
		protected void onPostExecute(Boolean success) {
			if (!success) {
				Log.w("VbofSendTask.onPostExecute", "failed a send");
			}
		}
	}

	private Network networkUtil;

	@Override
	protected void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);

		setContentView(R.layout.vbof);
		try {
			networkUtil = Network.getInstance(this);
		} catch (KeyStoreException e) {
			Log.e("PanopticonActivity.onCreate",
					"FATAL: exception loading the network utility");
			throw new RuntimeException("exception loading the network utility",
					e);
		}
	}

	@Override
	protected void onPause() {
		super.onPause();
	}

	@Override
	protected void onResume() {
		super.onResume();
		Intent intent = getIntent();
		new VbofSendTask().execute(intent);
	}
}
