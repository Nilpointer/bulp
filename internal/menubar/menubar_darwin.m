#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include "menubar.h"

static const CGFloat kWidth = 389; // same as the original popover

@interface MBController : NSObject <WKScriptMessageHandler, NSPopoverDelegate>
@property(strong) NSStatusItem *item;
@property(strong) NSPopover *popover;
@property(strong) WKWebView *web;
@property(copy) NSString *symbol;
@end

@implementation MBController

- (void)setupWithURL:(NSString *)url symbol:(NSString *)symbol {
	self.symbol = symbol;
	self.item = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
	[self setActive:NO];
	self.item.button.target = self;
	self.item.button.action = @selector(toggle:);

	WKWebViewConfiguration *cfg = [WKWebViewConfiguration new];
	// The page reports its height so the popover can size to fit.
	[cfg.userContentController addScriptMessageHandler:self name:@"resize"];
	self.web = [[WKWebView alloc] initWithFrame:NSMakeRect(0, 0, kWidth, 160) configuration:cfg];
	[self.web setValue:@NO forKey:@"drawsBackground"]; // let the popover's vibrancy show through
	[self.web loadRequest:[NSURLRequest requestWithURL:[NSURL URLWithString:url]]];

	NSViewController *vc = [NSViewController new];
	vc.view = self.web;
	self.popover = [NSPopover new];
	self.popover.contentViewController = vc;
	self.popover.contentSize = NSMakeSize(kWidth, 160);
	self.popover.behavior = NSPopoverBehaviorTransient;
	self.popover.delegate = self;
}

- (void)setActive:(BOOL)active {
	NSString *name = active ? [self.symbol stringByAppendingString:@".fill"] : self.symbol;
	NSImage *img = [NSImage imageWithSystemSymbolName:name accessibilityDescription:@"Lights"]
	               ?: [NSImage imageWithSystemSymbolName:self.symbol accessibilityDescription:@"Lights"];
	img.template = YES;
	self.item.button.image = img;
}

- (void)toggle:(id)sender {
	if (self.popover.shown) {
		[self.popover performClose:sender];
		return;
	}
	[NSApp activate];
	[self.popover showRelativeToRect:self.item.button.bounds ofView:self.item.button preferredEdge:NSRectEdgeMinY];
	[self.popover.contentViewController.view.window makeKeyWindow];
	[self.web evaluateJavaScript:@"window.onPopoverShow && window.onPopoverShow()" completionHandler:nil];
}

- (void)popoverDidClose:(NSNotification *)note {
	[self.web evaluateJavaScript:@"window.onPopoverHide && window.onPopoverHide()" completionHandler:nil];
}

- (void)userContentController:(WKUserContentController *)ucc didReceiveScriptMessage:(WKScriptMessage *)msg {
	if ([msg.name isEqualToString:@"resize"] && [msg.body isKindOfClass:[NSNumber class]]) {
		CGFloat h = MIN(MAX([msg.body doubleValue], 80), 720);
		self.popover.contentSize = NSMakeSize(kWidth, ceil(h));
	}
}

@end

static MBController *controller;

void mb_run(const char *url, const char *symbol) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory]; // no Dock icon
		controller = [MBController new];
		[controller setupWithURL:[NSString stringWithUTF8String:url]
		                  symbol:[NSString stringWithUTF8String:symbol]];
		[NSApp run];
	}
}

void mb_set_active(bool active) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[controller setActive:active];
	});
}

void mb_quit(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp terminate:nil];
	});
}
