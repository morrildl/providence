package dude.morrildl.providence.panopticon;

import android.content.Context;
import android.database.Cursor;
import android.support.v4.widget.SimpleCursorAdapter;

public class PanopticonCursorAdapter extends SimpleCursorAdapter {

	public PanopticonCursorAdapter(Context context, int layout, Cursor c,
			String[] from, int[] to, int flags) {
		super(context, layout, c, from, to, flags);
	}

}
