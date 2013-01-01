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
package dude.morrildl.providence.db;

import android.content.Context;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;

public class OpenHelper extends SQLiteOpenHelper {
	public static final String CREATE_EVENTS = "create table events (id integer primary key, which text not null, type text not null, event text not null, ts timestamp, eventid text not null)";
	public static final String CREATE_LAST_MOTION = "create table last_motion (which text not null primary key, ts timestamp)";
	public static final int DATABASE_VERSION = 3;
	public static final String REPLACE_LAST_MOTION = "replace into last_motion values (?, ?)";

	private static final String DATABASE_NAME = "providence";
	private static final String GC = "delete from events where ts < datetime('now', '-1week')";

	public OpenHelper(Context context) {
		super(context, DATABASE_NAME, null, DATABASE_VERSION);
	}

	@Override
	public synchronized SQLiteDatabase getWritableDatabase() {
		SQLiteDatabase db = super.getWritableDatabase();
		// run our housekeeping query (to prevent unbounded DB size)
		// before we hand over the connection to the caller
		db.execSQL(OpenHelper.GC);
		return db;
	}

	@Override
	public void onCreate(SQLiteDatabase db) {
		db.execSQL(CREATE_EVENTS);
		db.execSQL(CREATE_LAST_MOTION);
	}

	@Override
	public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
		if (oldVersion < 2) {
			db.execSQL("drop table events");
			db.execSQL(CREATE_EVENTS);
			db.execSQL(CREATE_LAST_MOTION);
		}
		if (oldVersion < 3) {
			db.execSQL("alter table events add column eventid text not null default '0'");
		}
	}
}
