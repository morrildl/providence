package dude.morrildl.lifer.providence;

import android.app.Fragment;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import dude.morrildl.lifer.R;

public class LatestEventFragment extends Fragment {
	@Override
	public View onCreateView(LayoutInflater inflater, ViewGroup container,
			Bundle savedInstanceState) {
		return inflater.inflate(R.layout.latest_event_fragment, container,
				false);
	}

	@Override
	public void onResume() {
		super.onResume();
		OpenHelper helper = new OpenHelper(this.getActivity());
		SQLiteDatabase db = helper.getReadableDatabase();
		Cursor c = db.query("events", new String[] { "which", "type", "event",
				"max(ts)" }, null, null, null, null, null);

		StringBuilder sb = new StringBuilder();
		c.moveToFirst();
		sb.append(c.getString(0));
		String s = c.getString(1);
		if (s != null && !"".equals(s)) {
			sb.append(" (a ").append(s).append(")");
		}
		sb.append(" reported '").append(c.getString(2))
				.append("' at ").append(c.getString(3));
		((TextView) getActivity().findViewById(
				R.id.latest_event_fragment_main_text)).setText(sb.toString());
	}
}
