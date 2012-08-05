package dude.morrildl.providence.panopticon;

import android.content.Context;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;

public class OpenHelper extends SQLiteOpenHelper {
	private static final String DATABASE_NAME = "providence";
	public static final int DATABASE_VERSION = 1;
	public static final String GC = "delete from events where ts < datetime('now', '-1week')";
	public static final String CREATE = "create table events (id integer auto_increment, which text not null, type text not null, event text not null, ts timestamp)";
	
	public OpenHelper(Context context) {
		super(context, DATABASE_NAME, null, DATABASE_VERSION);
	}

	@Override
	public void onCreate(SQLiteDatabase db) {
		db.execSQL(CREATE);
	}

	@Override
	public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
	}
}
