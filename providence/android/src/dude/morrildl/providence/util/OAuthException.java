package dude.morrildl.providence.util;

public class OAuthException extends Exception {
	public OAuthException(Exception e) {
		super(e);
	}

	public OAuthException(String string) {
		super(string);
	}

	private static final long serialVersionUID = 1L;
}
