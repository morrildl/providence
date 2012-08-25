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
package dude.morrildl.providence.photos;

import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;
import android.net.Uri;
import android.os.Handler;
import android.os.IBinder;
import android.os.Message;
import android.util.Log;

public class PhotoUploaderService extends Service {
	public static final String OPERATION = "dude.morrildl.providence.photoupload.operation";
	public static final int OPERATION_NOOP = 0;
	public static final int OPERATION_UPLOAD_PHOTO_URI = 1;
	public static final String URI_DATABASE_NAME = "dude.morrildl.providence";
	public static final int URI_DATABASE_VERSION = 1;

	private ProcessorThread processor = null;

	public static class UriDatabaseHelper extends SQLiteOpenHelper {
		public UriDatabaseHelper(Context context) {
			super(context, URI_DATABASE_NAME, null, URI_DATABASE_VERSION);
		}

		@Override
		public void onCreate(SQLiteDatabase db) {
			// create table
		}

		@Override
		public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
		}
	}

	private Handler stopHandler = null;

	private static class ProcessorThread extends Thread {
		private PhotoUploaderService service = null;
		private SQLiteDatabase database = null;

		@SuppressWarnings("unused")
		private ProcessorThread() {
		}

		public ProcessorThread(PhotoUploaderService service) {
			this.service = service;
		}

		private void openDatabase() {
			database = new UriDatabaseHelper(service).getWritableDatabase();
		}

		@Override
		public void run() {
			openDatabase();

			while (true) {
				try {
					Thread.sleep(5000);
					Log.d("PhotoUploaderService.ProcessorThread", "tick");
					break;
				} catch (Throwable t) {
					Log.e("PhotoUploaderService.ProcessorThread",
							"unhandled exception during processing", t);
				}
			}

			closeDatabase();
			service.halt();
		}

		private void closeDatabase() {
			if (database != null) {
				try {
					database.close();
				} catch (Throwable t) {
				}
			}
			database = null;
		}

		public void ping() {
			if (isAlive())
				return;
			start();
		}
	}

private SQLiteDatabase database = null;
	@Override
	public void onCreate() {
		super.onCreate();
		database = new UriDatabaseHelper(this).getWritableDatabase();
		processor = new ProcessorThread(this);
		stopHandler = new Handler() {
			@Override
			public void handleMessage(Message msg) {
				stopSelf();
			}
		};
	}

	private void addUriRow(Uri uri) {
		// insert
	}
	@Override
	public int onStartCommand(Intent intent, int flags, int startId) {
		super.onStartCommand(intent, flags, startId);
		if (intent.getExtras().getInt(OPERATION) == OPERATION_UPLOAD_PHOTO_URI) {
			Uri uri = intent.getData();
			addUriRow(uri);
			processor.ping();
		} else {
			Log.e("PhotoUploaderService.onStartCommand", "unknown operation "
					+ intent.getExtras().getInt(OPERATION));
		}
		return Service.START_STICKY;
	}

	@Override
	public IBinder onBind(Intent intent) {
		return null;
	}

	@Override
	public void onDestroy() {
		super.onDestroy();
		if (database != null) {
			try {
				database.close();
			} catch (Throwable t) {
			}
		}
	}

	public void halt() {
		stopHandler.sendEmptyMessage(42);
	}
}
