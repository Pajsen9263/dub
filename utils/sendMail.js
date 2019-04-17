const nodemailer = require( 'nodemailer' );

module.exports.sendMail = async function ( email, subject, content, attachment ) {
  /** Simple send email helper to minimise code. See nodemailer documentation for more details **/
  const transporter = nodemailer.createTransport( {
    service: 'gmail',
    auth: {
      user: process.env.EMAIL_NOREPLY,
      pass: process.env.PASSWORD_NOREPLY
    }
  } );

  const mail = {
    from: 'Digital Ungdom <' + process.env.EMAIL + '>',
    to: email,
    subject: subject,
    html: content,
  };

  if ( attachment ) mail[ 'attachments' ] = attachment;

  return await transporter.sendMail( mail );

};