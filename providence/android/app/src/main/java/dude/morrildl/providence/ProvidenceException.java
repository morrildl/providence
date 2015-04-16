package dude.morrildl.providence;

public class ProvidenceException extends Exception {
	private static final long serialVersionUID = -5649340928890927666L;

	public ProvidenceException() {
		super();
	}

	public ProvidenceException(String msg) {
		super(msg);
	}

	public ProvidenceException(String msg, Throwable t) {
		super(msg, t);
	}
}
