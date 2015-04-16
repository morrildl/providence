package dude.morrildl.providence;

public class OAuthException extends Exception {
	private static final long serialVersionUID = 1L;

	public OAuthException(String string) {
		super(string);
	}

	public OAuthException(Throwable t) {
		super(t);
	}
}
